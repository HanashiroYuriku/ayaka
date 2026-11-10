package port

import (
	"context"
	"time"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
)

// UserRepository is the persistence contract for users.
//
// Implementations must return apperror.ErrNotFound (or an error matching it)
// when a record does not exist, and apperror.ErrConflict on unique violations.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	List(ctx context.Context, page model.PageRequest) ([]entity.User, int64, error)
	Create(ctx context.Context, user *entity.User) error

	// UpdatePassword sets a new password hash and clears any login lockout.
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	MarkEmailVerified(ctx context.Context, id string, at time.Time) error
	// RegisterFailedLogin atomically increments the failed-login counter. When it
	// reaches maxAttempts the account is locked until lockUntil and the counter resets.
	RegisterFailedLogin(ctx context.Context, id string, maxAttempts int, lockUntil time.Time) error
	ResetFailedLogins(ctx context.Context, id string) error
}

// RefreshTokenRepository stores refresh tokens (by hash).
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	// FindByHash returns the token; with forUpdate it locks the row until the
	// surrounding transaction ends (prevents two concurrent rotations).
	FindByHash(ctx context.Context, hash string, forUpdate bool) (*entity.RefreshToken, error)
	Revoke(ctx context.Context, id string, replacedBy *string, at time.Time) error
	RevokeFamily(ctx context.Context, familyID string, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
	// DeleteStale removes tokens that expired or were revoked before the given time.
	DeleteStale(ctx context.Context, before time.Time) (int64, error)
}

// UserTokenRepository stores single-use email tokens (by hash).
type UserTokenRepository interface {
	Create(ctx context.Context, token *entity.UserToken) error
	FindByHash(ctx context.Context, purpose entity.TokenPurpose, hash string) (*entity.UserToken, error)
	// MarkUsed redeems the token. It must fail with apperror.ErrNotFound when the
	// token was already used, so that a token can be redeemed exactly once.
	MarkUsed(ctx context.Context, id string, at time.Time) error
	// InvalidateAll marks every unused token of the user for purpose as used.
	InvalidateAll(ctx context.Context, userID string, purpose entity.TokenPurpose, at time.Time) error
	// LastCreatedAt returns when the newest token of the user for purpose was
	// created; ok is false when there is none.
	LastCreatedAt(ctx context.Context, userID string, purpose entity.TokenPurpose) (t time.Time, ok bool, err error)
	DeleteStale(ctx context.Context, before time.Time) (int64, error)
}
