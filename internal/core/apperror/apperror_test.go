package apperror_test

import (
	"errors"
	"fmt"
	"testing"

	"be-ayaka/internal/core/apperror"

	"github.com/stretchr/testify/assert"
)

func TestIs_MatchesByKind(t *testing.T) {
	err := apperror.NotFound("user not found")

	assert.ErrorIs(t, err, apperror.ErrNotFound)
	assert.NotErrorIs(t, err, apperror.ErrConflict)
}

func TestIs_MatchesSpecificSentinel(t *testing.T) {
	sentinel := apperror.Unauthorized("invalid email or password")
	other := apperror.Unauthorized("token expired")

	wrapped := fmt.Errorf("login: %w", sentinel)

	assert.ErrorIs(t, wrapped, sentinel)
	assert.NotErrorIs(t, other, sentinel)
}

func TestWrap_KeepsCauseButHidesItPublicly(t *testing.T) {
	cause := errors.New("pq: connection refused")
	err := apperror.ErrInternal.Wrap(cause)

	assert.ErrorIs(t, err, cause)
	assert.Equal(t, "internal server error", err.PublicMessage())
	assert.Contains(t, err.Error(), "connection refused")
}

func TestWithFields_DoesNotMutateSentinel(t *testing.T) {
	err := apperror.ErrValidation.WithFields(map[string]string{"email": "invalid"})

	assert.Equal(t, "invalid", err.Fields["email"])
	assert.Nil(t, apperror.ErrValidation.Fields)
}

func TestAs_UnknownErrorBecomesInternal(t *testing.T) {
	err := apperror.As(errors.New("boom"))

	assert.Equal(t, apperror.KindInternal, err.Kind)
	assert.Equal(t, "internal server error", err.PublicMessage())
}
