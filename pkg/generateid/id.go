// Package generateid creates prefixed, time-ordered identifiers.
package generateid

import (
	"strings"

	"github.com/google/uuid"
)

// New returns an ID such as "USR-01932c7e6f2a7c3b9f1d2e3a4b5c6d7e".
//
// It uses UUIDv7, which is time-ordered and therefore friendly to B-tree
// indexes (unlike random UUIDv4 primary keys).
func New(prefix string) string {
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	clean := strings.ReplaceAll(id.String(), "-", "")

	if prefix == "" {
		prefix = "ID"
	}
	return strings.ToUpper(prefix) + "-" + clean
}
