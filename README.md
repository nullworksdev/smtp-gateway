# smtp-gateway

A lightweight Go service that exposes a single REST endpoint to send emails through **any caller-supplied SMTP server**. No credentials are stored — every request is self-contained.

---

## Quick start

```bash
go run .           # starts on :8080
PORT=9000 go run . # custom port
```

Or with Docker:

```bash
docker build -t smtp-gateway .
docker run -p 8080:8080 smtp-gateway
```

---

## API

### `POST /api/send`

Send an email.

#### Request body (JSON)

| Field       | Type       | Required | Description                                                 |
|-------------|------------|----------|-------------------------------------------------------------|
| `smtp_host` | `string`   | ✓        | SMTP server hostname, e.g. `smtp.gmail.com`                 |
| `smtp_port` | `integer`  | ✓        | `587` (STARTTLS), `465` (implicit TLS), or `25`             |
| `username`  | `string`   | ✓        | SMTP authentication username                                |
| `password`  | `string`   | ✓        | SMTP authentication password / app password                 |
| `from`      | `string`   | ✓        | Sender, e.g. `"No-Reply <no-reply@example.com>"`            |
| `to`        | `[]string` | ✓        | One or more recipient addresses                             |
| `cc`        | `[]string` |          | Optional CC recipients                                      |
| `bcc`       | `[]string` |          | Optional BCC recipients (excluded from message headers)     |
| `subject`   | `string`   | ✓        | Email subject line                                          |
| `body`      | `string`   | ✓*       | Plain-text body (* required if `html_body` is absent)       |
| `html_body` | `string`   |          | HTML body; if set, sends `multipart/alternative`            |

#### Success response — `200 OK`

```json
{
  "success": true,
  "message": "email sent successfully"
}
```

#### Validation error — `422 Unprocessable Entity`

```json
{
  "error": "validation failed",
  "details": {
    "smtp_port": "must be between 1 and 65535 (got 0)",
    "to": "at least one recipient is required"
  }
}
```

#### Gateway error — `502 Bad Gateway`

```json
{
  "error": "auth: 535 5.7.8 Username and Password not accepted"
}
```

---

### `GET /api/health`

```json
{ "status": "ok" }
```

---

## Example — plain-text email

```bash
curl -s -X POST http://localhost:8080/api/send \
  -H 'Content-Type: application/json' \
  -d '{
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "username":  "you@gmail.com",
    "password":  "your-app-password",
    "from":      "You <you@gmail.com>",
    "to":        ["alice@example.com"],
    "subject":   "Hello from smtp-gateway",
    "body":      "This is a plain-text test email."
  }'
```

## Example — multipart email with HTML

```bash
curl -s -X POST http://localhost:8080/api/send \
  -H 'Content-Type: application/json' \
  -d '{
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "username":  "you@gmail.com",
    "password":  "your-app-password",
    "from":      "No-Reply <no-reply@example.com>",
    "to":        ["alice@example.com", "bob@example.com"],
    "cc":        ["manager@example.com"],
    "subject":   "Welcome!",
    "body":      "Welcome to our service.",
    "html_body": "<h1>Welcome!</h1><p>Thanks for signing up.</p>"
  }'
```

---

## Notes

- **Port 587** — STARTTLS (recommended for most providers).
- **Port 465** — Implicit TLS (SMTPS); used by some providers.
- **Port 25** — Unauthenticated / smtp (dev/internal use only).
- Gmail requires an **App Password** (2FA must be enabled).
- Credentials are never logged or persisted.