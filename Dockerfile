FROM golang:1.26-alpine AS builder
 
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o smtp-gateway .
 
# --- final stage ---
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/smtp-gateway /smtp-gateway
EXPOSE 8080
ENTRYPOINT ["/smtp-gateway"]
 