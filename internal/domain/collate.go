package domain

import (
	"sort"
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

var (
	collatorOnce sync.Once
	collator     *collate.Collator
	collatorLock sync.Mutex
)

func CompareText(a, b string) int {
	collatorOnce.Do(func() {
		collator = collate.New(language.Und)
	})
	collatorLock.Lock()
	defer collatorLock.Unlock()
	return collator.CompareString(a, b)
}

func OrderBy[T any](items []T, descending bool, compare func(a, b T) int) {
	sort.SliceStable(items, func(i, j int) bool {
		result := compare(items[i], items[j])
		if descending {
			return result > 0
		}
		return result < 0
	})
}

func CompareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func CompareFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func CompareNumber(a, b *Number) int {
	left, right := NumberFromInt(0), NumberFromInt(0)
	if a != nil {
		left = *a
	}
	if b != nil {
		right = *b
	}
	return left.Decimal.Cmp(right.Decimal)
}
