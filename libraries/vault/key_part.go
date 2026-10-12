package vault

import (
	"time"

	"github.com/google/orderedcode"
)

type KeyPart[A any] struct {
	appendTo   func([]byte, A) []byte
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
	return part.appendTo(nil, value)
}

var TextKeyPart = KeyPart[string]{
	appendTo: appendOrderedText[string],
	holder: func() ([]any, func() (string, error)) {
		var text string

		return []any{&text}, func() (string, error) { return text, nil }
	},
}

var TextKeyPartDescending = descendingFrom(TextKeyPart)

func TextKeyPartFrom[A any](text func(A) string, valueOf func(string) (A, error)) KeyPart[A] {
	return KeyPart[A]{
		appendTo: func(encoded []byte, value A) []byte {
			return appendOrderedText(encoded, text(value))
		},
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
		appendTo: func(encoded []byte, value A) []byte {
			return appendOrderedText(encoded, byteForm(value))
		},
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
	appendTo: func(encoded []byte, instant time.Time) []byte {
		return appendOrderedInteger(encoded, instant.Unix())
	},
	holder: func() ([]any, func() (time.Time, error)) {
		var seconds int64

		return []any{&seconds}, func() (time.Time, error) {
			return time.Unix(seconds, 0).UTC(), nil
		}
	},
}

var TimeKeyPartDescending = descendingFrom(TimeKeyPart)

var IntegerKeyPart = KeyPart[int64]{
	appendTo: appendOrderedInteger,
	holder: func() ([]any, func() (int64, error)) {
		var number int64

		return []any{&number}, func() (int64, error) { return number, nil }
	},
}

var IntegerKeyPartDescending = descendingFrom(IntegerKeyPart)

func descendingFrom[A any](ascending KeyPart[A]) KeyPart[A] {
	return KeyPart[A]{
		appendTo: func(encoded []byte, value A) []byte {
			start := len(encoded)
			encoded = ascending.appendTo(encoded, value)
			invert(encoded[start:])

			return encoded
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
