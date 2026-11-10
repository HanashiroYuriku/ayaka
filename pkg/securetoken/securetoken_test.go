package securetoken_test

import (
	"testing"

	"be-ayaka/pkg/securetoken"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tok1, hash1, err := securetoken.New()
	require.NoError(t, err)
	tok2, _, err := securetoken.New()
	require.NoError(t, err)

	assert.Len(t, tok1, 43, "32 bytes in unpadded base64url")
	assert.Len(t, hash1, 64, "hex sha256")
	assert.NotEqual(t, tok1, tok2)
	assert.Equal(t, hash1, securetoken.Hash(tok1))
	assert.NotContains(t, tok1, "+")
	assert.NotContains(t, tok1, "/")
}
