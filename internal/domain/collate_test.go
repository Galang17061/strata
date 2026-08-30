package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderByKeepsStableOrderAndFlipsDirection(t *testing.T) {
	names := []string{"banana", "Apple", "cherry", "apple"}
	OrderBy(names, false, CompareText)
	assert.Equal(t, []string{"apple", "Apple", "banana", "cherry"}, names)
	OrderBy(names, true, CompareText)
	assert.Equal(t, "cherry", names[0])
}

func TestCompareNumberTreatsMissingAsZero(t *testing.T) {
	half := NumberFromInt(1)
	assert.Equal(t, -1, CompareNumber(nil, &half))
	assert.Equal(t, 0, CompareNumber(nil, nil))
}
