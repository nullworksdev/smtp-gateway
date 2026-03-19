package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"smtp-gateway/models"
)

var emailRE = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

// Validate checks all required fields of a SendRequest and returns a map of
// field → error message for every problem found, or nil if the request is valid.
func Validate(req *models.SendRequest) map[string]string {
	errs := make(map[string]string)

	// --- SMTP connection fields ---
	if strings.TrimSpace(req.SMTPHost) == "" {
		errs["smtp_host"] = "required"
	} else if _, err := net.LookupHost(req.SMTPHost); err != nil {
		// Non-fatal in case of test/local hosts; just validate non-empty above.
		_ = err
	}

	if req.SMTPPort <= 0 || req.SMTPPort > 65535 {
		errs["smtp_port"] = fmt.Sprintf("must be between 1 and 65535 (got %d)", req.SMTPPort)
	}

	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "required"
	}

	if strings.TrimSpace(req.Password) == "" {
		errs["password"] = "required"
	}

	// --- Message fields ---
	if strings.TrimSpace(req.From) == "" {
		errs["from"] = "required"
	} else if !containsEmail(req.From) {
		errs["from"] = "must contain a valid email address"
	}

	if len(req.To) == 0 {
		errs["to"] = "at least one recipient is required"
	} else {
		for i, addr := range req.To {
			if !containsEmail(addr) {
				errs[fmt.Sprintf("to[%d]", i)] = fmt.Sprintf("%q is not a valid address", addr)
			}
		}
	}

	for i, addr := range req.CC {
		if !containsEmail(addr) {
			errs[fmt.Sprintf("cc[%d]", i)] = fmt.Sprintf("%q is not a valid address", addr)
		}
	}

	for i, addr := range req.BCC {
		if !containsEmail(addr) {
			errs[fmt.Sprintf("bcc[%d]", i)] = fmt.Sprintf("%q is not a valid address", addr)
		}
	}

	if strings.TrimSpace(req.Subject) == "" {
		errs["subject"] = "required"
	}

	if strings.TrimSpace(req.Body) == "" && strings.TrimSpace(req.HTMLBody) == "" {
		errs["body"] = "either body or html_body is required"
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// containsEmail reports whether s contains a valid-looking email address,
// accounting for the "Display Name <addr>" format.
func containsEmail(s string) bool {
	return emailRE.MatchString(s)
}
