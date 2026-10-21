package vault

import (
	"time"
)

type KeyPart[A any] struct {
	appendTo   func([]byte, A) []byte
	parseFrom  func(orderedBytes) (A, orderedBytes, error)
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

func (part KeyPart[A]) keysBetween(lowest, highest A) KeyRange {
	lowestEncoding, highestEncoding := part.encodingOf(lowest), part.encodingOf(highest)

	if part.descending {
		return KeyRange{firstIncluded: highestEncoding, firstExcluded: successorOf(lowestEncoding)}
	}

	return KeyRange{firstIncluded: lowestEncoding, firstExcluded: successorOf(highestEncoding)}
}

func (part KeyPart[A]) encodingOf(value A) []byte {
	return part.appendTo(nil, value)
}

var TextKeyPart = KeyPart[string]{
	appendTo: appendOrderedText[string],
	parseFrom: func(stored orderedBytes) (string, orderedBytes, error) {
		text, rest, err := parseOrderedText(stored)

		return string(text), rest, err
	},
}

var TextKeyPartDescending = descendingFrom(TextKeyPart)

func TextKeyPartFrom[A any](text func(A) string, valueOf func(string) (A, error)) KeyPart[A] {
	return KeyPart[A]{
		appendTo: func(encoded []byte, value A) []byte {
			return appendOrderedText(encoded, text(value))
		},
		parseFrom: func(stored orderedBytes) (A, orderedBytes, error) {
			parsedText, rest, err := parseOrderedText(stored)
			if err != nil {
				var unparsed A

				return unparsed, rest, err
			}
			value, err := valueOf(string(parsedText))

			return value, rest, err
		},
	}
}

func BytesKeyPartFrom[A any](
	appendBytes func(A, []byte) []byte,
	valueOf func(StoredBytes) (A, error),
) KeyPart[A] {
	return KeyPart[A]{
		appendTo: func(encoded []byte, value A) []byte {
			return escapedTextFrom(appendBytes(value, encoded), len(encoded))
		},
		parseFrom: func(stored orderedBytes) (A, orderedBytes, error) {
			storedBytes, rest, err := parseOrderedText(stored)
			if err != nil {
				var unparsed A

				return unparsed, rest, err
			}
			value, err := valueOf(StoredBytes{stored: storedBytes})

			return value, rest, err
		},
	}
}

var TimeInSecondsKeyPart = KeyPart[time.Time]{
	appendTo: func(encoded []byte, instant time.Time) []byte {
		return appendOrderedInteger(encoded, instant.Unix())
	},
	parseFrom: func(stored orderedBytes) (time.Time, orderedBytes, error) {
		seconds, rest, err := parseOrderedInteger(stored)

		return time.Unix(seconds, 0).UTC(), rest, err
	},
}

var TimeInSecondsKeyPartDescending = descendingFrom(TimeInSecondsKeyPart)

var TimeInMillisecondsKeyPart = KeyPart[time.Time]{
	appendTo: func(encoded []byte, instant time.Time) []byte {
		return appendOrderedInteger(encoded, instant.UnixMilli())
	},
	parseFrom: func(stored orderedBytes) (time.Time, orderedBytes, error) {
		milliseconds, rest, err := parseOrderedInteger(stored)

		return time.UnixMilli(milliseconds).UTC(), rest, err
	},
}

var TimeInMillisecondsKeyPartDescending = descendingFrom(TimeInMillisecondsKeyPart)

var IntegerKeyPart = KeyPart[int64]{
	appendTo:  appendOrderedInteger,
	parseFrom: parseOrderedInteger,
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
		parseFrom: func(stored orderedBytes) (A, orderedBytes, error) {
			value, rest, err := ascending.parseFrom(stored.flipped())

			return value, rest.flipped(), err
		},
		descending: true,
	}
}
