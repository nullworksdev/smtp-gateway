package mailer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"smtp-gateway/models"
)

const (
	dialTimeout = 10 * time.Second
	sendTimeout = 30 * time.Second
)

// Send relays the email described by req through the caller-supplied SMTP server.
func Send(req *models.SendRequest) error {
	addr := fmt.Sprintf("%s:%d", req.SMTPHost, req.SMTPPort)

	// Build the raw RFC 2822 message bytes.
	msg, err := buildMessage(req)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	// Collect all envelope recipients (To + CC + BCC).
	allRecipients := collectRecipients(req.To, req.CC, req.BCC)
	if len(allRecipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	auth := smtp.PlainAuth("", req.Username, req.Password, req.SMTPHost)

	// Try STARTTLS first (port 587 / explicit TLS), fall back to implicit TLS
	// (port 465), then plain (port 25 / dev use).
	switch req.SMTPPort {
	case 465:
		return sendImplicitTLS(addr, auth, req.From, allRecipients, msg)
	default:
		return sendSTARTTLS(addr, auth, req.From, allRecipients, msg)
	}
}

// sendSTARTTLS dials plaintext and upgrades via STARTTLS (ports 25, 587, etc.)
func sendSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)

	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	// Attempt STARTTLS; some servers (e.g. local test servers) may not support it.
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	return relay(client, from, to, msg)
}

// sendImplicitTLS wraps the connection in TLS immediately (port 465 / SMTPS).
func sendImplicitTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)

	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	return relay(client, from, to, msg)
}

// relay performs the MAIL FROM / RCPT TO / DATA exchange.
func relay(client *smtp.Client, from string, to []string, msg []byte) error {
	envelopeFrom := extractAddress(from)
	if err := client.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(extractAddress(recipient)); err != nil {
			return fmt.Errorf("RCPT TO <%s>: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	defer w.Close()

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// buildMessage constructs a minimal but valid RFC 2822 email message.
// If HTMLBody is provided it builds a multipart/alternative message;
// otherwise it sends a plain-text message.
func buildMessage(req *models.SendRequest) ([]byte, error) {
	var buf bytes.Buffer

	writeHeader := func(key, value string) {
		fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
	}

	writeHeader("From", req.From)
	writeHeader("To", strings.Join(req.To, ", "))
	if len(req.CC) > 0 {
		writeHeader("Cc", strings.Join(req.CC, ", "))
	}
	writeHeader("Subject", encodeSubject(req.Subject))
	writeHeader("Date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	writeHeader("MIME-Version", "1.0")

	if req.HTMLBody != "" {
		boundary := "=_RelayBoundary_" + fmt.Sprintf("%d", time.Now().UnixNano())
		writeHeader("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary))
		buf.WriteString("\r\n")

		// Plain-text part
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		buf.WriteString(encodeBase64Body(req.Body))

		// HTML part
		fmt.Fprintf(&buf, "\r\n--%s\r\n", boundary)
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		buf.WriteString(encodeBase64Body(req.HTMLBody))

		fmt.Fprintf(&buf, "\r\n--%s--\r\n", boundary)
	} else {
		writeHeader("Content-Type", "text/plain; charset=UTF-8")
		writeHeader("Content-Transfer-Encoding", "base64")
		buf.WriteString("\r\n")
		buf.WriteString(encodeBase64Body(req.Body))
	}

	return buf.Bytes(), nil
}

// encodeBase64Body encodes the body using MIME base64 line wrapping.
func encodeBase64Body(s string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	// Wrap at 76 characters per MIME spec.
	var sb strings.Builder
	for len(encoded) > 76 {
		sb.WriteString(encoded[:76])
		sb.WriteString("\r\n")
		encoded = encoded[76:]
	}
	sb.WriteString(encoded)
	sb.WriteString("\r\n")
	return sb.String()
}

// encodeSubject encodes the subject as UTF-8 Base64 per RFC 2047 so non-ASCII
// characters are transported safely.
func encodeSubject(subject string) string {
	// Only encode if the subject contains non-ASCII characters.
	for _, r := range subject {
		if r > 127 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
		}
	}
	return subject
}

// extractAddress pulls the bare email address out of strings like
// "Alice <alice@example.com>" — returning just "alice@example.com".
func extractAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if start := strings.Index(addr, "<"); start != -1 {
		if end := strings.Index(addr, ">"); end > start {
			return addr[start+1 : end]
		}
	}
	return addr
}

// collectRecipients merges multiple recipient slices, deduplicating by address.
func collectRecipients(slices ...[]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, s := range slices {
		for _, addr := range s {
			bare := extractAddress(addr)
			bare = strings.ToLower(strings.TrimSpace(bare))
			if bare == "" {
				continue
			}
			if _, ok := seen[bare]; !ok {
				seen[bare] = struct{}{}
				result = append(result, addr)
			}
		}
	}
	return result
}
