package model

import (
	"time"

	"be-ayaka/internal/core/entity"
)

// TokenClaims is the verified identity carried by an access token.
type TokenClaims struct {
	TokenID   string
	UserID    string
	Role      entity.Role
	SessionID string // refresh token family the access token belongs to
	ExpiresAt time.Time
}

// AuthTokens is the result of a successful login or refresh.
type AuthTokens struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// ClientInfo describes the device that opened a session (stored for auditing).
type ClientInfo struct {
	IP        string
	UserAgent string
}

type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type ResetPasswordInput struct {
	Token       string
	NewPassword string
}
