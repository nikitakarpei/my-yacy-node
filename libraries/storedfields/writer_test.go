package storedfields_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
)

var errMalformed = errors.New("malformed record")

func TestEveryFieldReadsBackTheValueWritten(t *testing.T) {
	written := time.Date(2026, time.March, 4, 5, 6, 7, 8, time.UTC)

	var writer storedfields.Writer
	writer.Byte(0x7F)
	writer.Presence(true)
	writer.Uint16(65535)
	writer.Count(math.MaxInt)
	writer.Varint(math.MaxInt64)
	writer.Float(52.52)
	writer.Time(written)
	writer.Fixed([]byte("ABCD"))
	writer.Bytes([]byte{0x00, 0x01})
	writer.Text("a text, with a comma")

	reader := storedfields.ReaderOf(writer.Record(), errMalformed)
	if number := reader.Byte("byte"); number != 0x7F {
		t.Errorf("byte = %#x, want 0x7F", number)
	}
	if present := reader.Presence("presence"); !present {
		t.Error("presence = false, want true")
	}
	if number := reader.Uint16("uint16"); number != 65535 {
		t.Errorf("uint16 = %d, want 65535", number)
	}
	if number := reader.Count("count"); number != math.MaxInt {
		t.Errorf("count = %d, want %d", number, math.MaxInt)
	}
	if number := reader.Varint("varint"); number != math.MaxInt64 {
		t.Errorf("varint = %d, want %d", number, int64(math.MaxInt64))
	}
	if number := reader.Float("float"); number != 52.52 {
		t.Errorf("float = %v, want 52.52", number)
	}
	if instant := reader.Time("time"); !instant.Equal(written) {
		t.Errorf("time = %v, want %v", instant, written)
	}
	if value := string(reader.Fixed("fixed", 4)); value != "ABCD" {
		t.Errorf("fixed = %q, want \"ABCD\"", value)
	}
	if value := reader.Bytes("bytes"); len(value) != 2 || value[0] != 0x00 || value[1] != 0x01 {
		t.Errorf("bytes = %v, want [0 1]", value)
	}
	if value := reader.Text("text"); value != "a text, with a comma" {
		t.Errorf("text = %q, want \"a text, with a comma\"", value)
	}
	if err := reader.Err(); err != nil {
		t.Errorf("Err = %v, want no error", err)
	}
}

func TestACountLargerThanTheLargestCountIsMalformed(t *testing.T) {
	twoToThe63 := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}

	reader := storedfields.ReaderOf(twoToThe63, errMalformed)
	if number := reader.Count("count"); number != 0 {
		t.Errorf("count = %d, want 0", number)
	}
	if err := reader.Err(); !errors.Is(err, errMalformed) {
		t.Errorf("Err = %v, want errMalformed", err)
	}
}

func TestAnAbsentFieldReadsBackAbsent(t *testing.T) {
	var writer storedfields.Writer
	writer.Presence(false)

	reader := storedfields.ReaderOf(writer.Record(), errMalformed)
	if present := reader.Presence("presence"); present {
		t.Error("presence = true, want false")
	}
	if err := reader.Err(); err != nil {
		t.Errorf("Err = %v, want no error", err)
	}
}
