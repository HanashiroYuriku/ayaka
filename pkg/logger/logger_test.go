package logger_test

import (
	"context"
	"log/slog"
	"testing"

	"be-ayaka/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	l, err := logger.ParseLevel("warn")
	require.NoError(t, err)
	assert.Equal(t, slog.LevelWarn, l)

	l, err = logger.ParseLevel("")
	require.NoError(t, err)
	assert.Equal(t, slog.LevelInfo, l)

	_, err = logger.ParseLevel("loud")
	assert.Error(t, err)
}

func TestNew_RejectsUnknownFormat(t *testing.T) {
	_, _, err := logger.New(logger.Options{Format: "xml"})
	assert.Error(t, err)
}

func TestContextRoundTrip(t *testing.T) {
	l := slog.New(slog.DiscardHandler)

	ctx := logger.WithContext(context.Background(), l)

	assert.Same(t, l, logger.FromContext(ctx))
	assert.Same(t, slog.Default(), logger.FromContext(context.Background()))
}
