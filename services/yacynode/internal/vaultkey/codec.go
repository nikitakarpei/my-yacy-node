package vaultkey

import (
	"time"

	"github.com/google/orderedcode"
)

type Codec[A any] struct {
	items  func(A) []any
	holder func() ([]any, func() A)
}

var Text = Codec[string]{
	items: func(text string) []any { return []any{text} },
	holder: func() ([]any, func() string) {
		var text string

		return []any{&text}, func() string { return text }
	},
}

var TextDescending = descendingFrom(Text)

var Time = Codec[time.Time]{
	items: func(instant time.Time) []any {
		return []any{instant.Unix(), int64(instant.Nanosecond())}
	},
	holder: func() ([]any, func() time.Time) {
		var seconds, nanoseconds int64

		return []any{&seconds, &nanoseconds}, func() time.Time {
			return time.Unix(seconds, nanoseconds).UTC()
		}
	},
}

var TimeDescending = descendingFrom(Time)

var Integer = Codec[int64]{
	items: func(number int64) []any { return []any{number} },
	holder: func() ([]any, func() int64) {
		var number int64

		return []any{&number}, func() int64 { return number }
	},
}

var IntegerDescending = descendingFrom(Integer)

func descendingFrom[A any](ascending Codec[A]) Codec[A] {
	return Codec[A]{
		items: func(value A) []any {
			return descendingItemsFrom(ascending.items(value))
		},
		holder: func() ([]any, func() A) {
			targets, value := ascending.holder()

			return descendingItemsFrom(targets), value
		},
	}
}

func descendingItemsFrom(ascendingItems []any) []any {
	descendingItems := make([]any, len(ascendingItems))
	for position, item := range ascendingItems {
		descendingItems[position] = orderedcode.Decr(item)
	}

	return descendingItems
}
