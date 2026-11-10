package token

import (
	"testing"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProvider() *JWTProvider {
	return NewJWTProvider(Options{
		Issuer: "ayaka-test",
		Secret: "access-secret-for-tests-only-0123456789",
		TTL:    15 * time.Minute,
	})
}

func TestIssueAndParse(t *testing.T) {
	p := newProvider()

	tok, exp, err := p.Issue("USR-1", entity.RoleAdmin, "SES-1")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), exp, 2*time.Second)

	claims, err := p.Parse(tok)
	require.NoError(t, err)
	assert.Equal(t, "USR-1", claims.UserID)
	assert.Equal(t, entity.RoleAdmin, claims.Role)
	assert.Equal(t, "SES-1", claims.SessionID)
	assert.NotEmpty(t, claims.TokenID)
}

func TestParse_Expired(t *testing.T) {
	p := newProvider()
	p.now = func() time.Time { return time.Now().Add(-time.Hour) }
	tok, _, err := p.Issue("USR-1", entity.RoleUser, "SES-1")
	require.NoError(t, err)

	p.now = time.Now
	_, err = p.Parse(tok)
	require.ErrorIs(t, err, apperror.ErrUnauthorized)
	assert.Equal(t, "token expired", apperror.As(err).PublicMessage())
}

func TestParse_RejectsTamperedForeignAndWrongType(t *testing.T) {
	p := newProvider()
	tok, _, err := p.Issue("USR-1", entity.RoleUser, "SES-1")
	require.NoError(t, err)

	_, err = p.Parse(tok + "x")
	assert.ErrorIs(t, err, apperror.ErrUnauthorized, "tampered signature")

	other := NewJWTProvider(Options{Issuer: "someone-else", Secret: "another-secret-another-secret-0000", TTL: time.Minute})
	foreign, _, err := other.Issue("USR-1", entity.RoleAdmin, "SES-1")
	require.NoError(t, err)
	_, err = p.Parse(foreign)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized, "foreign issuer/secret")

	// A correctly signed token that is not an access token must be rejected.
	now := time.Now()
	wrongType, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "USR-1", Issuer: "ayaka-test",
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
	}).SignedString([]byte(p.opts.Secret))
	require.NoError(t, err)
	_, err = p.Parse(wrongType)
	assert.ErrorIs(t, err, apperror.ErrUnauthorized, "wrong token type")
}
