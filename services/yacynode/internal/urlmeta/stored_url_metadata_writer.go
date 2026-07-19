package urlmeta

import (
	"bytes"
	"encoding/binary"
	"math"
)

type urlMetadataWriter struct {
	buf bytes.Buffer
}

func (w *urlMetadataWriter) uint8(number byte) {
	w.buf.WriteByte(number)
}

func (w *urlMetadataWriter) count(number int) {
	w.varint(int64(number))
}

func (w *urlMetadataWriter) varint(number int64) {
	var tmp [binary.MaxVarintLen64]byte
	written := binary.PutVarint(tmp[:], number)
	w.buf.Write(tmp[:written])
}

func (w *urlMetadataWriter) degrees(value float64) {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], math.Float64bits(value))
	w.buf.Write(tmp[:])
}

func (w *urlMetadataWriter) lengthPrefixed(data []byte) {
	var tmp [binary.MaxVarintLen64]byte
	written := binary.PutUvarint(tmp[:], uint64(len(data)))
	w.buf.Write(tmp[:written])
	w.buf.Write(data)
}

func (w *urlMetadataWriter) text(value string) {
	w.lengthPrefixed([]byte(value))
}

func (w *urlMetadataWriter) bytes() []byte {
	return w.buf.Bytes()
}
