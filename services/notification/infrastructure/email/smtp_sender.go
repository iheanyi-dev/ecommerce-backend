package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	notification "github.com/iheanyi-dev/ecommerce-backend/services/notification"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
)

type SMTPSender struct {
	config *notification.Config
}

func NewSMTPSender(config *notification.Config) (*SMTPSender, error) {
	if config == nil {
		return nil, fmt.Errorf("SMTP config is required")
	}

	if strings.TrimSpace(config.SMTPHost) == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}

	if config.SMTPPort < 1 || config.SMTPPort > 65535 {
		return nil, fmt.Errorf("invalid SMTP port")
	}

	if strings.TrimSpace(config.SMTPFrom) == "" {
		return nil, fmt.Errorf("SMTP sender address is required")
	}

	if (config.SMTPUsername == "") != (config.SMTPPassword == "") {
		return nil, fmt.Errorf("SMTP username and password must both be provided")
	}

	return &SMTPSender{config: config}, nil
}

func (s *SMTPSender) Send(ctx context.Context, email entities.Email) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	from, err := mail.ParseAddress(s.config.SMTPFrom)
	if err != nil {
		return fmt.Errorf("invalid SMTP sender address: %w", err)
	}

	to, err := mail.ParseAddress(email.Recipient().Value())
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	subject := email.Subject()
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("email subject contains invalid characters")
	}

	message := buildMessage(from.Address, to.Address, subject, email)

	address := net.JoinHostPort(
		s.config.SMTPHost,
		strconv.Itoa(s.config.SMTPPort),
	)

	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("start SMTP TLS: %w", err)
	}

	if s.config.SMTPUsername != "" {
		auth := smtp.PlainAuth(
			"",
			s.config.SMTPUsername,
			s.config.SMTPPassword,
			s.config.SMTPHost,
		)

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}

	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}

	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}

	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}

	return nil
}

func buildMessage(from, to, subject string, email entities.Email) []byte {
	var message strings.Builder

	message.WriteString("From: " + from + "\r\n")
	message.WriteString("To: " + to + "\r\n")
	message.WriteString("Subject: " + subject + "\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: " + email.Format().Value() + "; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	message.WriteString("\r\n")
	message.WriteString(email.Body())

	return []byte(message.String())
}

// BuildMessageForTest exposes message formatting for package-level tests.
func BuildMessageForTest(from, to, subject string, email entities.Email) []byte {
	return buildMessage(from, to, subject, email)
}
