package port

import (
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Compare returns nil when password matches hash.
	Compare(hash, password string) error
}

// TokenProvider issues and verifies authentication tokens.
//
// Parse must return an error matching apperror.ErrUnauthorized when the token
// is malformed, expired, signed with the wrong key, or of the wrong type.
type TokenProvider interface {
	Issue(userID string, role entity.Role) (*model.AuthTokens, error)
	Parse(token string, typ model.TokenType) (*model.TokenClaims, error)
}
