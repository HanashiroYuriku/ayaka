// Package token implements port.TokenProvider using signed JWTs (HS256).
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

// Options configures the JWT provider.
type Options struct {
	Issuer        string
	AccessSecret  string
	AccessTTL     time.Duration
	RefreshSecret string
	RefreshTTL    time.Duration
}

type claims struct {
	Role entity.Role     `json:"role"`
	Type model.TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// JWTProvider issues access/refresh token pairs signed with separate secrets.
type JWTProvider struct {
	opts Options
	now  func() time.Time
}

func NewJWTProvider(opts Options) *JWTProvider {
	return &JWTProvider{opts: opts, now: time.Now}
}

func (p *JWTProvider) Issue(userID string, role entity.Role) (*model.AuthTokens, error) {
	now := p.now()

	accessExp := now.Add(p.opts.AccessTTL)
	access, err := p.sign(userID, role, model.TokenTypeAccess, now, accessExp, p.opts.AccessSecret)
	if err != nil {
		return nil, err
	}

	refreshExp := now.Add(p.opts.RefreshTTL)
	refresh, err := p.sign(userID, role, model.TokenTypeRefresh, now, refreshExp, p.opts.RefreshSecret)
	if err != nil {
		return nil, err
	}

	return &model.AuthTokens{
		AccessToken:      access,
		RefreshToken:     refresh,
		TokenType:        "Bearer",
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (p *JWTProvider) Parse(tokenStr string, typ model.TokenType) (*model.TokenClaims, error) {
	secret := p.opts.AccessSecret
	if typ == model.TokenTypeRefresh {
		secret = p.opts.RefreshSecret
	}

	c := &claims{}
	_, err := jwt.ParseWithClaims(tokenStr, c,
		func(*jwt.Token) (any, error) { return []byte(secret), nil },
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

	if c.Type != typ {
		return nil, apperror.Unauthorized("invalid token type")
	}
	if c.Subject == "" {
		return nil, apperror.Unauthorized("invalid token subject")
	}

	return &model.TokenClaims{
		TokenID:   c.ID,
		UserID:    c.Subject,
		Role:      c.Role,
		Type:      c.Type,
		ExpiresAt: c.ExpiresAt.Time,
	}, nil
}

func (p *JWTProvider) sign(userID string, role entity.Role, typ model.TokenType, now, exp time.Time, secret string) (string, error) {
	c := claims{
		Role: role,
		Type: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			Issuer:    p.opts.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign %s token: %w", typ, err)
	}
	return signed, nil
}
