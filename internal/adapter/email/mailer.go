package email

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"sync"
	"time"

	"be-ayaka/internal/core/port"
	"be-ayaka/pkg/logger"
)

//go:embed templates/*.html
var templateFS embed.FS

// TemplateMailer implements port.AuthMailer by rendering embedded HTML
// templates and delivering them through a port.EmailSender.
type TemplateMailer struct {
	appName string
	sender  port.EmailSender
	verify  *template.Template
	reset   *template.Template
}

func NewTemplateMailer(appName string, sender port.EmailSender) (*TemplateMailer, error) {
	parse := func(name string) (*template.Template, error) {
		return template.ParseFS(templateFS, "templates/layout.html", "templates/"+name)
	}
	verify, err := parse("verify_email.html")
	if err != nil {
		return nil, fmt.Errorf("parse email template: %w", err)
	}
	reset, err := parse("reset_password.html")
	if err != nil {
		return nil, fmt.Errorf("parse email template: %w", err)
	}
	return &TemplateMailer{appName: appName, sender: sender, verify: verify, reset: reset}, nil
}

type templateData struct {
	AppName    string
	Subject    string
	Name       string
	Link       string
	ButtonText string
	ValidFor   string
}

func (m *TemplateMailer) SendEmailVerification(ctx context.Context, to, name, link string, validFor time.Duration) error {
	return m.send(ctx, m.verify, to, templateData{
		Subject: "Confirm your email address", Name: name, Link: link,
		ButtonText: "Verify email", ValidFor: humanDuration(validFor),
	})
}

func (m *TemplateMailer) SendPasswordReset(ctx context.Context, to, name, link string, validFor time.Duration) error {
	return m.send(ctx, m.reset, to, templateData{
		Subject: "Reset your password", Name: name, Link: link,
		ButtonText: "Reset password", ValidFor: humanDuration(validFor),
	})
}

func (m *TemplateMailer) send(ctx context.Context, t *template.Template, to string, data templateData) error {
	data.AppName = m.appName
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		return fmt.Errorf("render email: %w", err)
	}
	return m.sender.SendEmail(ctx, to, data.Subject, buf.String())
}

func humanDuration(d time.Duration) string {
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		return fmt.Sprintf("%d hour(s)", int(d.Hours()))
	case d >= time.Minute:
		return fmt.Sprintf("%d minute(s)", int(d.Minutes()))
	default:
		return d.String()
	}
}

// ---------------------------------------------------------------- senders

// LogSender writes emails to the log instead of sending them. Development only:
// it prints the full body, including one-time links.
type LogSender struct {
	log *slog.Logger
}

func NewLogSender(log *slog.Logger) *LogSender { return &LogSender{log: log} }

func (s *LogSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	logger.FromContext(ctx).Info("email (log driver, not sent)",
		slog.String("to", to), slog.String("subject", subject), slog.String("body", htmlBody))
	return nil
}

// AsyncSender sends emails in the background so that request latency does not
// depend on the SMTP server — and does not reveal whether an account exists.
// Call Wait during shutdown to let in-flight emails finish.
type AsyncSender struct {
	next    port.EmailSender
	timeout time.Duration
	wg      sync.WaitGroup
}

func NewAsyncSender(next port.EmailSender, timeout time.Duration) *AsyncSender {
	return &AsyncSender{next: next, timeout: timeout}
}

func (s *AsyncSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	// Keep context values (request-scoped logger) but not the cancellation:
	// the request finishes before the email is sent.
	bg := context.WithoutCancel(ctx)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		sendCtx, cancel := context.WithTimeout(bg, s.timeout)
		defer cancel()
		if err := s.next.SendEmail(sendCtx, to, subject, htmlBody); err != nil {
			logger.FromContext(bg).Error("send email failed", slog.String("subject", subject), slog.Any("error", err))
		}
	}()
	return nil
}

// Wait blocks until pending emails are sent or ctx is done.
func (s *AsyncSender) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("pending emails not sent: %w", ctx.Err())
	}
}
