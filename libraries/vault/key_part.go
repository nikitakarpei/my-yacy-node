package vault

import (
	"time"

	"github.com/google/orderedcode"
)

type KeyPart[A any] struct {
	items      func(A) []any
	holder     func() ([]any, func() (A, error))
	descending bool
}

func (part KeyPart[A]) keysWith(value A) KeyRange {
	encoding := part.encodingOf(value)

	return KeyRange{firstIncluded: encoding, firstExcluded: successorOf(encoding)}
}

func (part KeyPart[A]) keysFrom(value A) KeyRange {
	encoding := part.encodingOf(value)

	if part.descending {
		return KeyRange{firstExcluded: successorOf(encoding)}
	}

	return KeyRange{firstIncluded: encoding}
}

func (part KeyPart[A]) keysThrough(value A) KeyRange {
	encoding := part.encodingOf(value)

	if part.descending {
		return KeyRange{firstIncluded: encoding}
	}

	return KeyRange{firstExcluded: successorOf(encoding)}
}

func (part KeyPart[A]) keysBefore(value A) KeyRange {
	encoding := part.encodingOf(value)

	if part.descending {
		return KeyRange{firstIncluded: successorOf(encoding)}
	}

	return KeyRange{firstExcluded: encoding}
}

func (part KeyPart[A]) encodingOf(value A) []byte {
	return keyOf(part.items(value)).encoded
}

var TextKeyPart = KeyPart[string]{
	items: func(text string) []any { return []any{text} },
	holder: func() ([]any, func() (string, error)) {
		var text string

		return []any{&text}, func() (string, error) { return text, nil }
	},
}

var TextKeyPartDescending = descendingFrom(TextKeyPart)

func TextKeyPartFrom[A any](text func(A) string, valueOf func(string) (A, error)) KeyPart[A] {
	return KeyPart[A]{
		items: func(value A) []any { return TextKeyPart.items(text(value)) },
		holder: func() ([]any, func() (A, error)) {
			targets, textInKey := TextKeyPart.holder()

			return targets, func() (A, error) {
				parsedText, err := textInKey()
				if err != nil {
					var unparsed A

					return unparsed, err
				}

				return valueOf(parsedText)
			}
		},
		descending: TextKeyPart.descending,
	}
}

func BytesKeyPartFrom[A any](
	byteForm func(A) []byte,
	valueOf func([]byte) (A, error),
) KeyPart[A] {
	return KeyPart[A]{
		items: func(value A) []any { return TextKeyPart.items(string(byteForm(value))) },
		holder: func() ([]any, func() (A, error)) {
			targets, textInKey := TextKeyPart.holder()

			return targets, func() (A, error) {
				storedBytes, err := textInKey()
				if err != nil {
					var unparsed A

					return unparsed, err
				}

				return valueOf([]byte(storedBytes))
			}
		},
		descending: TextKeyPart.descending,
	}
}

var TimeKeyPart = KeyPart[time.Time]{
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

var TimeKeyPartDescending = descendingFrom(TimeKeyPart)

var IntegerKeyPart = KeyPart[int64]{
	items: func(number int64) []any { return []any{number} },
	holder: func() ([]any, func() (int64, error)) {
		var number int64

		return []any{&number}, func() (int64, error) { return number, nil }
	},
}

var IntegerKeyPartDescending = descendingFrom(IntegerKeyPart)

func descendingFrom[A any](ascending KeyPart[A]) KeyPart[A] {
	return KeyPart[A]{
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
