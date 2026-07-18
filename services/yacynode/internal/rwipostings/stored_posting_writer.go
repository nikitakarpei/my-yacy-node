package rwipostings

import (
	"bytes"
	"encoding/binary"
)

type postingWriter struct {
	buf bytes.Buffer
}

func (w *postingWriter) uint8(number byte) {
	w.buf.WriteByte(number)
}

func (w *postingWriter) uint16(number uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], number)
	w.buf.Write(tmp[:])
}

func (w *postingWriter) count(number int) {
	w.varint(int64(number))
}

func (w *postingWriter) varint(number int64) {
	var tmp [binary.MaxVarintLen64]byte
	written := binary.PutVarint(tmp[:], number)
	w.buf.Write(tmp[:written])
}

func (w *postingWriter) fixed(data []byte) {
	w.buf.Write(data)
}

func (w *postingWriter) lengthPrefixed(data []byte) {
	var tmp [binary.MaxVarintLen64]byte
	written := binary.PutUvarint(tmp[:], uint64(len(data)))
	w.buf.Write(tmp[:written])
	w.buf.Write(data)
}

func (w *postingWriter) bytes() []byte {
	return w.buf.Bytes()
}
