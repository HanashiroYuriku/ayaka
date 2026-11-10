package entity

import "time"

// RefreshToken is one refresh token of a login session.
//
// Only the SHA-256 hash of the token is stored. Every refresh rotates the
// token: the old one is revoked (ReplacedBy points to the new one) and a new
// one is created in the same Family. Presenting a revoked token means it was
// stolen or replayed, so the whole family (session) is revoked.
type RefreshToken struct {
	ID         string     `gorm:"primaryKey;type:varchar(50)"`
	UserID     string     `gorm:"type:varchar(50);not null;index"`
	FamilyID   string     `gorm:"type:varchar(50);not null;index"`
	TokenHash  string     `gorm:"type:varchar(64);not null;uniqueIndex"`
	ExpiresAt  time.Time  `gorm:"not null"`
	RevokedAt  *time.Time `gorm:""`
	ReplacedBy *string    `gorm:"type:varchar(50)"`
	UserAgent  string     `gorm:"type:varchar(255)"`
	IP         string     `gorm:"type:varchar(64)"`
	CreatedAt  time.Time  `gorm:"not null"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }

func (t *RefreshToken) IsExpired(now time.Time) bool { return !now.Before(t.ExpiresAt) }

// TokenPurpose identifies what a one-time user token can be used for.
type TokenPurpose string

const (
	PurposePasswordReset     TokenPurpose = "password_reset"
	PurposeEmailVerification TokenPurpose = "email_verification"
)

// UserToken is a single-use token sent by email (password reset, email
// verification). Only its SHA-256 hash is stored.
type UserToken struct {
	ID        string       `gorm:"primaryKey;type:varchar(50)"`
	UserID    string       `gorm:"type:varchar(50);not null;index"`
	Purpose   TokenPurpose `gorm:"type:varchar(32);not null"`
	TokenHash string       `gorm:"type:varchar(64);not null;uniqueIndex"`
	ExpiresAt time.Time    `gorm:"not null"`
	UsedAt    *time.Time   `gorm:""`
	CreatedAt time.Time    `gorm:"not null"`
}

func (UserToken) TableName() string { return "user_tokens" }

// IsUsable reports whether the token can still be redeemed at time now.
func (t *UserToken) IsUsable(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}
