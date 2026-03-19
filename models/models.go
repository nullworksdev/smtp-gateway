package models

// SendRequest holds all parameters required to send an email via a user-supplied SMTP server.
type SendRequest struct {
	// SMTP connection details
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`

	// Message fields
	From     string   `json:"from"` // e.g. "Alice <alice@example.com>"
	To       []string `json:"to"`   // one or more recipients
	CC       []string `json:"cc"`   // optional CC recipients
	BCC      []string `json:"bcc"`  // optional BCC recipients
	Subject  string   `json:"subject"`
	Body     string   `json:"body"`
	HTMLBody string   `json:"html_body"` // optional; if set, sends multipart/alternative
}

// SendResponse is returned after a successful (or failed) relay attempt.
type SendResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ErrorResponse is returned when the request itself is invalid.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}
