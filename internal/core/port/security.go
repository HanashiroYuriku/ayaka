package port

import (
	"time"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Compare returns nil when password matches hash.
	Compare(hash, password string) error
}

// AccessTokenProvider issues and verifies short-lived access tokens.
//
// Parse must return an error matching apperror.ErrUnauthorized when the token
// is malformed, expired, signed with the wrong key, or not an access token.
type AccessTokenProvider interface {
	Issue(userID string, role entity.Role, sessionID string) (token string, expiresAt time.Time, err error)
	Parse(token string) (*model.TokenClaims, error)
}
