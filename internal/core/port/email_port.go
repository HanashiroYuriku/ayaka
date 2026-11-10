package port

import "context"

// EmailSender delivers transactional emails.
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, htmlBody string) error
}
