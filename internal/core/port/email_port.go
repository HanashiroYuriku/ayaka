package port

import (
	"context"
	"time"
)

// EmailSender delivers a raw HTML email.
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, htmlBody string) error
}

// AuthMailer sends the account emails used by the auth flows. Implementations
// own the templates; the core only provides the data.
type AuthMailer interface {
	SendEmailVerification(ctx context.Context, to, name, link string, validFor time.Duration) error
	SendPasswordReset(ctx context.Context, to, name, link string, validFor time.Duration) error
}
