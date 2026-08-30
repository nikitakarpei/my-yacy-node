package storedfields

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"
)

type Writer struct {
	record bytes.Buffer
}

func (w *Writer) Byte(number byte) {
	w.record.WriteByte(number)
}

func (w *Writer) Presence(present bool) {
	if present {
		w.record.WriteByte(1)

		return
	}
	w.record.WriteByte(0)
}

func (w *Writer) Uint16(number uint16) {
	var field [2]byte
	binary.LittleEndian.PutUint16(field[:], number)
	w.record.Write(field[:])
}

func (w *Writer) Count(number int) {
	var field [binary.MaxVarintLen64]byte
	//nolint:gosec // a count is a quantity and is never negative.
	written := binary.PutUvarint(field[:], uint64(number))
	w.record.Write(field[:written])
}

func (w *Writer) Varint(number int64) {
	var field [binary.MaxVarintLen64]byte
	written := binary.PutVarint(field[:], number)
	w.record.Write(field[:written])
}

func (w *Writer) Float(number float64) {
	var field [8]byte
	binary.LittleEndian.PutUint64(field[:], math.Float64bits(number))
	w.record.Write(field[:])
}

func (w *Writer) Time(instant time.Time) {
	w.Varint(instant.Unix())
	w.Varint(int64(instant.Nanosecond()))
}

func (w *Writer) Fixed(data []byte) {
	w.record.Write(data)
}

func (w *Writer) Bytes(data []byte) {
	var length [binary.MaxVarintLen64]byte
	written := binary.PutUvarint(length[:], uint64(len(data)))
	w.record.Write(length[:written])
	w.record.Write(data)
}

func (w *Writer) Text(value string) {
	w.Bytes([]byte(value))
}

func (w *Writer) Record() []byte {
	return w.record.Bytes()
}
