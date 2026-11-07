package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// SMTPSender delivers account activation emails through an SMTP server.
type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     *mail.Address
}

// NewSMTPSender constructs an SMTP sender from Identity configuration.
func NewSMTPSender(cfg *config.Config) (*SMTPSender, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SMTP configuration is required")
	}

	host := strings.TrimSpace(cfg.SMTPHost)
	port := strings.TrimSpace(cfg.SMTPPort)
	if port == "" {
		port = "587"
	}

	if host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid SMTP port")
	}

	from, err := mail.ParseAddress(strings.TrimSpace(cfg.SMTPFrom))
	if err != nil || from.Address == "" {
		return nil, fmt.Errorf("valid SMTP sender address is required")
	}

	if (cfg.SMTPUsername == "") != (cfg.SMTPPassword == "") {
		return nil, fmt.Errorf("SMTP username and password must be configured together")
	}

	return &SMTPSender{
		host:     host,
		port:     port,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     from,
	}, nil
}

// SendActivationEmail sends an account activation link to the recipient.
func (s *SMTPSender) SendActivationEmail(
	ctx context.Context,
	recipient user.Email,
	activationURL string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	to, err := mail.ParseAddress(recipient.String())
	if err != nil {
		return fmt.Errorf("invalid activation email recipient: %w", err)
	}

	parsedURL, err := url.ParseRequestURI(activationURL)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "https" && parsedURL.Scheme != "http") {
		return fmt.Errorf("invalid activation URL")
	}

	address := net.JoinHostPort(s.host, s.port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	if s.port == "465" {
		conn, err = (&tls.Dialer{
			NetDialer: dialer,
			Config: &tls.Config{
				ServerName: s.host,
				MinVersion: tls.VersionTLS12,
			},
		}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	stopCancellation := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopCancellation()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()

	tlsActive := s.port == "465"
	if !tlsActive {
		if supported, _ := client.Extension("STARTTLS"); supported {
			if err := client.StartTLS(&tls.Config{
				ServerName: s.host,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				return fmt.Errorf("start SMTP TLS: %w", err)
			}
			tlsActive = true
		}
	}

	if s.username != "" {
		if !tlsActive {
			return fmt.Errorf("refusing SMTP authentication without TLS")
		}
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := client.Mail(s.from.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}

	message := strings.Join([]string{
		"From: " + s.from.String(),
		"To: " + to.String(),
		"Subject: Verify your email address",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Welcome!",
		"",
		"Please activate your account using the link below:",
		activationURL,
		"",
		"If you did not create this account, you can ignore this email.",
		"",
	}, "\r\n")

	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("send SMTP message: %w", err)
	}

	return client.Quit()
}

var _ interface {
	SendActivationEmail(context.Context, user.Email, string) error
} = (*SMTPSender)(nil)
