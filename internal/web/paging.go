package web

import "math"

func TotalPages(totalData, pageSize int) int {
	if pageSize == 0 {
		return math.MinInt32
	}
	return int(math.Ceil(float64(totalData) / float64(pageSize)))
}

func Slice[T any](items []T, page, pageSize int) []T {
	skip := (page - 1) * pageSize
	if skip < 0 {
		skip = 0
	}
	if skip > len(items) {
		skip = len(items)
	}
	take := pageSize
	if take < 0 {
		take = 0
	}
	end := skip + take
	if end > len(items) {
		end = len(items)
	}
	return items[skip:end]
}

func Page[T any](items []T, page, pageSize int) ([]T, *Meta) {
	items = NonNil(items)
	return Slice(items, page, pageSize), NewMeta(len(items), TotalPages(len(items), pageSize), page, pageSize)
}

func PageOptional[T any](items []T, page, pageSize *int) ([]T, *Meta) {
	items = NonNil(items)
	if page == nil || pageSize == nil {
		return items, NewMeta(len(items), 1, 1, len(items))
	}
	return Page(items, *page, *pageSize)
}

func NonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
