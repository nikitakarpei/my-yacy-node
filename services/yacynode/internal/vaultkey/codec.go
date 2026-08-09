package vaultkey

import (
	"time"

	"github.com/google/orderedcode"
)

type Codec[A any] struct {
	items      func(A) []any
	holder     func() ([]any, func() (A, error))
	descending bool
}

func (codec Codec[A]) keysWith(value A) KeyRange {
	encoding := codec.encodingOf(value)

	return KeyRange{firstIncluded: encoding, firstExcluded: successorOf(encoding)}
}

func (codec Codec[A]) keysFrom(value A) KeyRange {
	encoding := codec.encodingOf(value)

	if codec.descending {
		return KeyRange{firstExcluded: successorOf(encoding)}
	}

	return KeyRange{firstIncluded: encoding}
}

func (codec Codec[A]) keysThrough(value A) KeyRange {
	encoding := codec.encodingOf(value)

	if codec.descending {
		return KeyRange{firstIncluded: encoding}
	}

	return KeyRange{firstExcluded: successorOf(encoding)}
}

func (codec Codec[A]) keysBefore(value A) KeyRange {
	encoding := codec.encodingOf(value)

	if codec.descending {
		return KeyRange{firstIncluded: successorOf(encoding)}
	}

	return KeyRange{firstExcluded: encoding}
}

func (codec Codec[A]) encodingOf(value A) []byte {
	return keyOf(codec.items(value)).encoded
}

var Text = Codec[string]{
	items: func(text string) []any { return []any{text} },
	holder: func() ([]any, func() (string, error)) {
		var text string

		return []any{&text}, func() (string, error) { return text, nil }
	},
}

var TextDescending = descendingFrom(Text)

func TextAs[A any](text func(A) string, valueOf func(string) (A, error)) Codec[A] {
	return Codec[A]{
		items: func(value A) []any { return Text.items(text(value)) },
		holder: func() ([]any, func() (A, error)) {
			targets, textInKey := Text.holder()

			return targets, func() (A, error) {
				parsedText, err := textInKey()
				if err != nil {
					var unparsed A

					return unparsed, err
				}

				return valueOf(parsedText)
			}
		},
		descending: Text.descending,
	}
}

var Time = Codec[time.Time]{
	items: func(instant time.Time) []any {
		return []any{instant.Unix(), int64(instant.Nanosecond())}
	},
	holder: func() ([]any, func() (time.Time, error)) {
		var seconds, nanoseconds int64

		return []any{&seconds, &nanoseconds}, func() (time.Time, error) {
			return time.Unix(seconds, nanoseconds).UTC(), nil
		}
	},
}

var TimeDescending = descendingFrom(Time)

var Integer = Codec[int64]{
	items: func(number int64) []any { return []any{number} },
	holder: func() ([]any, func() (int64, error)) {
		var number int64

		return []any{&number}, func() (int64, error) { return number, nil }
	},
}

var IntegerDescending = descendingFrom(Integer)

func descendingFrom[A any](ascending Codec[A]) Codec[A] {
	return Codec[A]{
		items: func(value A) []any {
			return descendingItemsFrom(ascending.items(value))
		},
		holder: func() ([]any, func() (A, error)) {
			targets, value := ascending.holder()

			return descendingItemsFrom(targets), value
		},
		descending: true,
	}
}

func descendingItemsFrom(ascendingItems []any) []any {
	descendingItems := make([]any, len(ascendingItems))
	for position, item := range ascendingItems {
		descendingItems[position] = orderedcode.Decr(item)
	}

	return descendingItems
}
