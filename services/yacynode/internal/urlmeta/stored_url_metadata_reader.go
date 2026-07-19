package urlmeta

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataReader struct {
	reader *bytes.Reader
	err    error
}

func newURLMetadataReader(data []byte) *urlMetadataReader {
	return &urlMetadataReader{reader: bytes.NewReader(data)}
}

func (r *urlMetadataReader) fail(field string, cause error) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: read %s: %w", yacymodel.ErrBadURLMetadata, field, cause)
	}
}

func (r *urlMetadataReader) uint8(field string) byte {
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

func (r *urlMetadataReader) count(field string) int {
	return int(r.varint(field))
}

func (r *urlMetadataReader) varint(field string) int64 {
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

func (r *urlMetadataReader) degrees(field string) float64 {
	if r.err != nil {
		return 0
	}
	var tmp [8]byte
	if _, err := io.ReadFull(r.reader, tmp[:]); err != nil {
		r.fail(field, err)

		return 0
	}

	return math.Float64frombits(binary.LittleEndian.Uint64(tmp[:]))
}

func (r *urlMetadataReader) lengthPrefixed(field string) []byte {
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

func (r *urlMetadataReader) text(field string) string {
	return string(r.lengthPrefixed(field))
}
