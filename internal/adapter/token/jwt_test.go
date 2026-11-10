package token

import (
	"testing"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProvider() *JWTProvider {
	return NewJWTProvider(Options{
		Issuer:        "ayaka-test",
		AccessSecret:  "access-secret-for-tests-only-0123456789",
		AccessTTL:     15 * time.Minute,
		RefreshSecret: "refresh-secret-for-tests-only-0123456789",
		RefreshTTL:    24 * time.Hour,
	})
}

func TestIssueAndParse(t *testing.T) {
	p := newProvider()

	tokens, err := p.Issue("USR-1", entity.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, "Bearer", tokens.TokenType)

	access, err := p.Parse(tokens.AccessToken, model.TokenTypeAccess)
	require.NoError(t, err)
	assert.Equal(t, "USR-1", access.UserID)
	assert.Equal(t, entity.RoleAdmin, access.Role)
	assert.NotEmpty(t, access.TokenID)

	refresh, err := p.Parse(tokens.RefreshToken, model.TokenTypeRefresh)
	require.NoError(t, err)
	assert.Equal(t, model.TokenTypeRefresh, refresh.Type)
}

func TestParse_RejectsWrongTokenType(t *testing.T) {
	p := newProvider()
	tokens, err := p.Issue("USR-1", entity.RoleUser)
	require.NoError(t, err)

	// A refresh token must never be usable as an access token (and vice versa).
	_, err = p.Parse(tokens.RefreshToken, model.TokenTypeAccess)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized)

	_, err = p.Parse(tokens.AccessToken, model.TokenTypeRefresh)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized)
}

func TestParse_Expired(t *testing.T) {
	p := newProvider()
	p.now = func() time.Time { return time.Now().Add(-time.Hour) }
	tokens, err := p.Issue("USR-1", entity.RoleUser)
	require.NoError(t, err)

	p.now = time.Now
	_, err = p.Parse(tokens.AccessToken, model.TokenTypeAccess)
	require.ErrorIs(t, err, apperror.ErrUnauthorized)
	assert.Equal(t, "token expired", apperror.As(err).PublicMessage())
}

func TestParse_RejectsTamperedAndForeignTokens(t *testing.T) {
	p := newProvider()
	tokens, err := p.Issue("USR-1", entity.RoleUser)
	require.NoError(t, err)

	_, err = p.Parse(tokens.AccessToken+"x", model.TokenTypeAccess)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized)

	other := NewJWTProvider(Options{
		Issuer: "someone-else", AccessSecret: "another-secret-another-secret-0000",
		AccessTTL: time.Minute, RefreshSecret: "x", RefreshTTL: time.Minute,
	})
	foreign, err := other.Issue("USR-1", entity.RoleAdmin)
	require.NoError(t, err)
	_, err = p.Parse(foreign.AccessToken, model.TokenTypeAccess)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized)
}
