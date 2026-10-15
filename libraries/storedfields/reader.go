package storedfields

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"slices"
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
	var packed [1]byte
	if !r.readInto(field, packed[:]) {
		return 0
	}

	return packed[0]
}

func (r *Reader) Presence(field string) bool {
	return r.Byte(field) != 0
}

func (r *Reader) Uint16(field string) uint16 {
	var packed [2]byte
	if !r.readInto(field, packed[:]) {
		return 0
	}

	return binary.LittleEndian.Uint16(packed[:])
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
	var packed [8]byte
	if !r.readInto(field, packed[:]) {
		return 0
	}

	return math.Float64frombits(binary.LittleEndian.Uint64(packed[:]))
}

func (r *Reader) Time(field string) time.Time {
	if r.noFieldRemains() {
		return time.Time{}
	}
	if seconds := r.Varint(field); r.err == nil {
		return time.Unix(seconds, 0).UTC()
	}

	return time.Time{}
}

func (r *Reader) Fixed(field string, length int) []byte {
	if r.noFieldRemains() {
		return nil
	}
	value := make([]byte, length)
	if !r.readInto(field, value) {
		return nil
	}

	return value
}

func (r *Reader) readInto(field string, buffer []byte) bool {
	if r.noFieldRemains() {
		return false
	}
	if read, _ := r.record.Read(buffer); read < len(buffer) {
		r.fail(field, io.ErrUnexpectedEOF)

		return false
	}

	return true
}

func (r *Reader) Bytes(field string) []byte {
	length, found := r.lengthOf(field)
	if !found {
		return nil
	}

	return r.Fixed(field, length)
}

func (r *Reader) lengthOf(field string) (int, bool) {
	if r.noFieldRemains() {
		return 0, false
	}
	length, err := binary.ReadUvarint(r.record)
	if err != nil {
		r.fail(field, err)

		return 0, false
	}
	//nolint:gosec // the bytes left in a record are never negative.
	if length > uint64(r.record.Len()) {
		r.fail(field, fmt.Errorf(
			"length %d exceeds the %d bytes left", length, r.record.Len(),
		))

		return 0, false
	}

	//nolint:gosec // the length is bounded by the bytes left above.
	return int(length), true
}

func (r *Reader) AppendBytes(buffer []byte, field string) []byte {
	length, found := r.lengthOf(field)
	if !found {
		return buffer
	}
	start := len(buffer)
	buffer = slices.Grow(buffer, length)[:start+length]
	_, _ = r.record.Read(buffer[start:])

	return buffer
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
