// Package email implements port.EmailSender over SMTP.
package email

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPSender sends HTML emails through an SMTP server using STARTTLS.
type SMTPSender struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPSender(host, port, user, pass, from string) *SMTPSender {
	if from == "" {
		from = user
	}
	return &SMTPSender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTPSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("email: header injection attempt detected")
	}

	headers := []string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="UTF-8"`,
	}
	msg := strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody

	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	addr := net.JoinHostPort(s.host, s.port)

	// net/smtp is not context aware; run it in a goroutine so callers can cancel.
	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, s.user, []string{to}, []byte(msg)) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
