package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const messageIDBytes = 16

var (
	errPendingEntryTruncated = errors.New("pending page visit entry shorter than its header")
	errPendingEntryMalformed = errors.New("pending page visit entry has a malformed time")
)

type messageID [messageIDBytes]byte

type pendingEntry struct {
	due       int64
	arrival   int64
	messageID messageID
	body      []byte
}

type pendingEntryCodec struct{}

func (pendingEntryCodec) Encode(entry pendingEntry) ([]byte, error) {
	encoded := make([]byte, 0, 2*binary.MaxVarintLen64+messageIDBytes+len(entry.body))
	encoded = binary.AppendVarint(encoded, entry.due)
	encoded = binary.AppendVarint(encoded, entry.arrival)
	encoded = append(encoded, entry.messageID[:]...)
	return append(encoded, entry.body...), nil
}

func (pendingEntryCodec) Decode(stored []byte) (pendingEntry, error) {
	due, dueBytes := binary.Varint(stored)
	if dueBytes <= 0 {
		return pendingEntry{}, errPendingEntryMalformed
	}
	arrival, arrivalBytes := binary.Varint(stored[dueBytes:])
	if arrivalBytes <= 0 {
		return pendingEntry{}, errPendingEntryMalformed
	}
	rest := stored[dueBytes+arrivalBytes:]
	if len(rest) < messageIDBytes {
		return pendingEntry{}, fmt.Errorf("%w: %d bytes", errPendingEntryTruncated, len(stored))
	}
	entry := pendingEntry{
		due:     due,
		arrival: arrival,
		body:    append([]byte(nil), rest[messageIDBytes:]...),
	}
	copy(entry.messageID[:], rest[:messageIDBytes])
	return entry, nil
}
