package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// Sender adalah abstraksi pengiriman email. Implementasi produksi memakai SMTP
// (Brevo), sedangkan development memakai DevSender yang menulis ke log.
type Sender interface {
	Send(ctx context.Context, to, subject, text string) error
}

type SMTPSender struct {
	addr string
	auth smtp.Auth
	from string
}

func NewSMTPSender(host string, port int, user, pass, from string) *SMTPSender {
	addr := fmt.Sprintf("%s:%d", host, port)
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	return &SMTPSender{addr: addr, auth: auth, from: from}
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	msg := strings.Join([]string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		text,
	}, "\r\n")
	return smtp.SendMail(s.addr, s.auth, s.from, []string{to}, []byte(msg))
}

// DevSender menulis OTP ke log. Hanya boleh dipakai saat APP_ENV=development.
type DevSender struct{}

func (DevSender) Send(_ context.Context, to, subject, text string) error {
	slog.Info("email development", "to", to, "subject", subject, "body", text)
	return nil
}
