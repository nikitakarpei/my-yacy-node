package storedfields

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

type Reader struct {
	record    *bytes.Reader
	malformed error
	err       error
}

func ReaderOf(record []byte, malformed error) *Reader {
	return &Reader{record: bytes.NewReader(record), malformed: malformed}
}

func (r *Reader) Err() error {
	return r.err
}

func (r *Reader) BytesLeft() int {
	if r.err != nil {
		return 0
	}

	return r.record.Len()
}

func (r *Reader) Reject(field string, cause error) {
	r.fail(field, cause)
}

func (r *Reader) Byte(field string) byte {
	packed := r.Fixed(field, 1)
	if packed == nil {
		return 0
	}

	return packed[0]
}

func (r *Reader) Presence(field string) bool {
	return r.Byte(field) != 0
}

func (r *Reader) Uint16(field string) uint16 {
	packed := r.Fixed(field, 2)
	if packed == nil {
		return 0
	}

	return binary.LittleEndian.Uint16(packed)
}

func (r *Reader) Count(field string) int {
	if r.noFieldRemains() {
		return 0
	}
	number, err := binary.ReadUvarint(r.record)
	if err != nil {
		r.fail(field, err)

		return 0
	}
	if number > math.MaxInt {
		r.fail(field, fmt.Errorf("%d exceeds the largest count", number))

		return 0
	}

	return int(number)
}

func (r *Reader) Varint(field string) int64 {
	if r.noFieldRemains() {
		return 0
	}
	number, err := binary.ReadVarint(r.record)
	if err != nil {
		r.fail(field, err)

		return 0
	}

	return number
}

func (r *Reader) Float(field string) float64 {
	packed := r.Fixed(field, 8)
	if packed == nil {
		return 0
	}

	return math.Float64frombits(binary.LittleEndian.Uint64(packed))
}

func (r *Reader) Time(field string) time.Time {
	if r.noFieldRemains() {
		return time.Time{}
	}
	seconds := r.Varint(field + " seconds")
	nanoseconds := r.Varint(field + " nanoseconds")
	if r.err != nil {
		return time.Time{}
	}

	return time.Unix(seconds, nanoseconds).UTC()
}

func (r *Reader) Fixed(field string, length int) []byte {
	if r.noFieldRemains() {
		return nil
	}
	value := make([]byte, length)
	if _, err := io.ReadFull(r.record, value); err != nil {
		r.fail(field, err)

		return nil
	}

	return value
}

func (r *Reader) Bytes(field string) []byte {
	if r.noFieldRemains() {
		return nil
	}
	length, err := binary.ReadUvarint(r.record)
	if err != nil {
		r.fail(field, err)

		return nil
	}
	//nolint:gosec // the bytes left in a record are never negative.
	if length > uint64(r.record.Len()) {
		r.fail(field, fmt.Errorf(
			"length %d exceeds the %d bytes left", length, r.record.Len(),
		))

		return nil
	}

	//nolint:gosec // the length is bounded by the bytes left above.
	return r.Fixed(field, int(length))
}

func (r *Reader) Text(field string) string {
	return string(r.Bytes(field))
}

func (r *Reader) noFieldRemains() bool {
	return r.err != nil || r.record.Len() == 0
}

func (r *Reader) fail(field string, cause error) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: read %s: %w", r.malformed, field, cause)
	}
}
