// Package hash provides password hashing.
package hash

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// DefaultCost balances security and latency (~250ms on modern hardware).
const DefaultCost = 12

// Bcrypt hashes passwords with bcrypt. It satisfies port.PasswordHasher.
type Bcrypt struct {
	cost int
}

// NewBcrypt creates a bcrypt hasher. A cost of 0 uses DefaultCost.
func NewBcrypt(cost int) *Bcrypt {
	if cost == 0 {
		cost = DefaultCost
	}
	return &Bcrypt{cost: cost}
}

// Hash returns the bcrypt hash of password.
// bcrypt only supports up to 72 bytes; longer inputs return bcrypt.ErrPasswordTooLong.
func (b *Bcrypt) Hash(password string) (string, error) {
	out, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Compare returns nil when password matches the hash.
func (b *Bcrypt) Compare(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrMismatch
	}
	return err
}

var ErrMismatch = errors.New("password does not match")
