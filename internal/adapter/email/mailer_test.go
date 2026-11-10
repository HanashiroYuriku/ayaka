package email

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureSender struct {
	mu      sync.Mutex
	subject string
	body    string
	err     error
	delay   time.Duration
}

func (c *captureSender) SendEmail(_ context.Context, _, subject, body string) error {
	time.Sleep(c.delay)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subject, c.body = subject, body
	return c.err
}

func TestTemplateMailer_RendersAndEscapes(t *testing.T) {
	sender := &captureSender{}
	m, err := NewTemplateMailer("Ayaka", sender)
	require.NoError(t, err)

	err = m.SendPasswordReset(context.Background(), "riku@mail.com", "<b>Riku</b>",
		"http://localhost:3000/reset-password?token=abc", 30*time.Minute)
	require.NoError(t, err)

	assert.Equal(t, "Reset your password", sender.subject)
	assert.Contains(t, sender.body, "reset-password?token=abc")
	assert.Contains(t, sender.body, "30 minute(s)")
	assert.Contains(t, sender.body, "&lt;b&gt;Riku&lt;/b&gt;", "user input must be HTML-escaped")

	require.NoError(t, m.SendEmailVerification(context.Background(), "riku@mail.com", "Riku", "http://x/verify?token=t", 24*time.Hour))
	assert.Equal(t, "Confirm your email address", sender.subject)
	assert.Contains(t, sender.body, "24 hour(s)")
}

func TestAsyncSender_ReturnsImmediatelyAndWaits(t *testing.T) {
	next := &captureSender{delay: 50 * time.Millisecond, err: errors.New("smtp down")}
	s := NewAsyncSender(next, time.Second)

	start := time.Now()
	require.NoError(t, s.SendEmail(context.Background(), "a@b.c", "subject", "body"), "errors are logged, not returned")
	assert.Less(t, time.Since(start), 40*time.Millisecond)

	require.NoError(t, s.Wait(context.Background()))
	next.mu.Lock()
	defer next.mu.Unlock()
	assert.Equal(t, "subject", next.subject)
}

func TestAsyncSender_WaitHonoursDeadline(t *testing.T) {
	s := NewAsyncSender(&captureSender{delay: 200 * time.Millisecond}, time.Second)
	_ = s.SendEmail(context.Background(), "a@b.c", "s", "b")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	assert.Error(t, s.Wait(ctx))
	_ = s.Wait(context.Background())
}
