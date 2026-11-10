// Package securetoken generates opaque, URL-safe random tokens and the
// SHA-256 hashes that are stored instead of the tokens themselves.
package securetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// byteLength gives 256 bits of entropy.
const byteLength = 32

// New returns a random token and its hash. Send the token to the client and
// store only the hash.
func New() (token, hash string, err error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, Hash(token), nil
}

// Hash returns the hex-encoded SHA-256 of token. A fast hash is fine here
// because the token has 256 bits of entropy (unlike passwords).
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
