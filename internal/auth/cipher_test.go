package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCipherRoundTripsAndIsDeterministic(t *testing.T) {
	c := NewCipher("0J3wr3!9bd")
	first := c.Encrypt("admin123")
	second := c.Encrypt("admin123")
	assert.Equal(t, first, second)
	decrypted, err := c.Decrypt(first)
	require.NoError(t, err)
	assert.Equal(t, "admin123", decrypted)
}

func TestCipherMatchesReferenceOutput(t *testing.T) {
	c := NewCipher("0J3wr3!9bd")
	assert.Equal(t, "vvSae0QP/VQ09zLbVo4oig==", c.Encrypt("admin"))
	assert.Equal(t, "muN4v7anSk/uk4rnaC33q2n+H6mE8UjjBqBoWW2frZI=", c.Encrypt("admin123"))
}
