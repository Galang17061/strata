package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderedMapKeepsInsertionOrderInJson(t *testing.T) {
	m := NewOrderedMap()
	m.Set("zeta", 1)
	m.Set("alpha", nil)
	m.Set("zeta", 2)
	encoded, err := json.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, `{"zeta":2,"alpha":null}`, string(encoded))
	assert.Equal(t, []string{"zeta", "alpha"}, m.Keys())
	empty, err := json.Marshal(NewOrderedMap())
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(empty))
}
