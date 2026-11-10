// Package token implements port.AccessTokenProvider using signed JWTs (HS256).
//
// Only short-lived access tokens are JWTs. Refresh tokens are opaque random
// strings stored (hashed) in the database so they can be rotated and revoked.
package token

import (
	"errors"
	"fmt"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const accessTokenType = "access"

// Options configures the JWT provider.
type Options struct {
	Issuer string
	Secret string
	TTL    time.Duration
}

type claims struct {
	Role      entity.Role `json:"role"`
	Type      string      `json:"typ"`
	SessionID string      `json:"sid"`
	jwt.RegisteredClaims
}

// JWTProvider issues and verifies access tokens.
type JWTProvider struct {
	opts Options
	now  func() time.Time
}

func NewJWTProvider(opts Options) *JWTProvider {
	return &JWTProvider{opts: opts, now: time.Now}
}

func (p *JWTProvider) Issue(userID string, role entity.Role, sessionID string) (string, time.Time, error) {
	now := p.now()
	exp := now.Add(p.opts.TTL)

	c := claims{
		Role:      role,
		Type:      accessTokenType,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			Issuer:    p.opts.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(p.opts.Secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

func (p *JWTProvider) Parse(tokenStr string) (*model.TokenClaims, error) {
	c := &claims{}
	_, err := jwt.ParseWithClaims(tokenStr, c,
		func(*jwt.Token) (any, error) { return []byte(p.opts.Secret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(p.opts.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(p.now),
	)
	if err != nil {
		msg := "invalid token"
		if errors.Is(err, jwt.ErrTokenExpired) {
			msg = "token expired"
		}
		return nil, apperror.Unauthorized(msg).Wrap(err)
	}

	if c.Type != accessTokenType {
		return nil, apperror.Unauthorized("invalid token type")
	}
	if c.Subject == "" {
		return nil, apperror.Unauthorized("invalid token subject")
	}

	return &model.TokenClaims{
		TokenID:   c.ID,
		UserID:    c.Subject,
		Role:      c.Role,
		SessionID: c.SessionID,
		ExpiresAt: c.ExpiresAt.Time,
	}, nil
}
