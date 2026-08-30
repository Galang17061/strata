package web

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageSlicesLikeSkipTake(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	page, meta := Page(items, 2, 2)
	assert.Equal(t, []int{3, 4}, page)
	assert.Equal(t, 3, meta.TotalPage)
	assert.True(t, meta.HasNextPage)
	beyond, _ := Page(items, 9, 2)
	assert.Equal(t, []int{}, beyond)
	negative, meta := Page(items, 0, -1)
	assert.Equal(t, []int{}, negative)
	assert.Equal(t, -5, meta.TotalPage)
	_, zero := Page(items, 1, 0)
	assert.Equal(t, math.MinInt32, zero.TotalPage)
}

func TestPageOptionalReturnsEverythingWithoutPaging(t *testing.T) {
	var none []string
	all, meta := PageOptional(none, nil, nil)
	assert.Equal(t, []string{}, all)
	assert.Equal(t, NewMeta(0, 1, 1, 0), meta)
}
