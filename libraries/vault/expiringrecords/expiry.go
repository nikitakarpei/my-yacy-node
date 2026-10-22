package expiringrecords

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

const expiryLength = 4

var (
	errExpiryOutOfRange = errors.New("expiry time is before 1970 or after February 2106")
	errExpiryMissing    = errors.New("stored record is shorter than its expiry time")
)

type expiry uint32

func expiryFrom(expiresAt time.Time) (expiry, error) {
	seconds := expiresAt.Unix()
	if expiresAt.Nanosecond() > 0 {
		seconds++
	}
	if seconds < 0 || seconds > math.MaxUint32 {
		return 0, fmt.Errorf("%w: %s", errExpiryOutOfRange, expiresAt)
	}

	return expiry(seconds), nil
}

func (e expiry) hasPassedAt(at time.Time) bool {
	return !time.Unix(int64(e), 0).After(at)
}

func storedFrom(storedExpiry expiry, record []byte) []byte {
	stored := make([]byte, 0, expiryLength+len(record))
	stored = binary.BigEndian.AppendUint32(stored, uint32(storedExpiry))

	return append(stored, record...)
}

func expiryAndRecordOf(stored []byte) (expiry, []byte, error) {
	if len(stored) < expiryLength {
		return 0, nil, fmt.Errorf("%w: %d bytes", errExpiryMissing, len(stored))
	}

	return expiry(binary.BigEndian.Uint32(stored)), stored[expiryLength:], nil
}
