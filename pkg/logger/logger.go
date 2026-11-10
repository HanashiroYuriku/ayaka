// Package logger builds the application's structured logger (log/slog) and
// carries request-scoped loggers through context.Context.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Options configures the logger.
type Options struct {
	Level  string // debug | info | warn | error
	Format string // json | text
	// Optional rotating log file, in addition to stdout.
	FilePath   string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
	// Static attributes added to every record (app name, version, env...).
	Attrs []slog.Attr
}

// New creates a logger. The returned io.Closer flushes/closes the log file (if any).
func New(opts Options) (*slog.Logger, io.Closer, error) {
	level, err := ParseLevel(opts.Level)
	if err != nil {
		return nil, nil, err
	}

	var (
		w      io.Writer = os.Stdout
		closer io.Closer = nopCloser{}
	)
	if opts.FilePath != "" {
		lj := &lumberjack.Logger{
			Filename:   opts.FilePath,
			MaxSize:    opts.MaxSizeMB,
			MaxBackups: opts.MaxBackups,
			MaxAge:     opts.MaxAgeDays,
			Compress:   opts.Compress,
		}
		w = io.MultiWriter(os.Stdout, lj)
		closer = lj
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var h slog.Handler
	switch strings.ToLower(opts.Format) {
	case "", "json":
		h = slog.NewJSONHandler(w, handlerOpts)
	case "text":
		h = slog.NewTextHandler(w, handlerOpts)
	default:
		return nil, nil, fmt.Errorf("unknown log format %q (want json|text)", opts.Format)
	}

	if len(opts.Attrs) > 0 {
		h = h.WithAttrs(opts.Attrs)
	}
	return slog.New(h), closer, nil
}

// ParseLevel converts a level name into a slog.Level.
func ParseLevel(s string) (slog.Level, error) {
	var l slog.Level
	if s == "" {
		return slog.LevelInfo, nil
	}
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("invalid log level %q: %w", s, err)
	}
	return l, nil
}

type ctxKey struct{}

// WithContext returns a copy of ctx that carries l.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger stored in ctx, or slog.Default().
// Request-scoped loggers already contain request_id (and user_id once authenticated).
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
