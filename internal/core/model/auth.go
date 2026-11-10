package model

import (
	"time"

	"be-ayaka/internal/core/entity"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// TokenClaims is the verified identity carried by a token.
type TokenClaims struct {
	TokenID   string
	UserID    string
	Role      entity.Role
	Type      TokenType
	ExpiresAt time.Time
}

// AuthTokens is the result of a successful authentication.
type AuthTokens struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}
