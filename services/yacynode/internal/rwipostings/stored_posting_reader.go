package rwipostings

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingReader struct {
	reader *bytes.Reader
	err    error
}

func newPostingReader(data []byte) *postingReader {
	return &postingReader{reader: bytes.NewReader(data)}
}

func (r *postingReader) fail(field string, cause error) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: read %s: %w", yacymodel.ErrBadRWIPosting, field, cause)
	}
}

func (r *postingReader) uint8(field string) byte {
	if r.err != nil {
		return 0
	}
	number, err := r.reader.ReadByte()
	if err != nil {
		r.fail(field, err)
		return 0
	}
	return number
}

func (r *postingReader) uint16(field string) uint16 {
	if r.err != nil {
		return 0
	}
	var tmp [2]byte
	if _, err := io.ReadFull(r.reader, tmp[:]); err != nil {
		r.fail(field, err)
		return 0
	}
	return binary.LittleEndian.Uint16(tmp[:])
}

func (r *postingReader) varint(field string) int64 {
	if r.err != nil {
		return 0
	}
	number, err := binary.ReadVarint(r.reader)
	if err != nil {
		r.fail(field, err)
		return 0
	}
	return number
}

func (r *postingReader) fixed(field string, length int) []byte {
	if r.err != nil {
		return nil
	}
	raw := make([]byte, length)
	if _, err := io.ReadFull(r.reader, raw); err != nil {
		r.fail(field, err)
		return nil
	}
	return raw
}

func (r *postingReader) lengthPrefixed(field string) []byte {
	if r.err != nil {
		return nil
	}
	length, err := binary.ReadUvarint(r.reader)
	if err != nil {
		r.fail(field, err)
		return nil
	}
	if remaining := r.reader.Len(); remaining < 0 || length > uint64(remaining) {
		r.fail(field, fmt.Errorf("length %d exceeds remaining bytes", length))
		return nil
	}
	raw := make([]byte, length)
	if _, err := io.ReadFull(r.reader, raw); err != nil {
		r.fail(field, err)
		return nil
	}
	return raw
}
