package storedfields_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
)

func TestFieldsAppendedAfterAValueWasStoredReadAsZero(t *testing.T) {
	var stored storedfields.Writer
	stored.Text("an address")

	reader := storedfields.ReaderOf(stored.Record(), errMalformed)
	if address := reader.Text("address"); address != "an address" {
		t.Errorf("address = %q, want \"an address\"", address)
	}
	if snippet := reader.Text("snippet"); snippet != "" {
		t.Errorf("snippet = %q, want the zero value", snippet)
	}
	if words := reader.Count("word count"); words != 0 {
		t.Errorf("word count = %d, want the zero value", words)
	}
	if hash := reader.Fixed("favicon hash", 12); hash != nil {
		t.Errorf("favicon hash = %v, want the zero value", hash)
	}
	if err := reader.Err(); err != nil {
		t.Errorf("Err = %v, want no error", err)
	}
}

func TestARecordCutInsideAFieldIsMalformed(t *testing.T) {
	var stored storedfields.Writer
	stored.Text("an address that the record does not carry in full")

	reader := storedfields.ReaderOf(stored.Record()[:6], errMalformed)
	reader.Text("address")

	if err := reader.Err(); !errors.Is(err, errMalformed) {
		t.Errorf("Err = %v, want errMalformed", err)
	}
}

func TestAFieldCutShortOfItsWidthIsMalformed(t *testing.T) {
	var stored storedfields.Writer
	stored.Float(52.52)

	for _, testCase := range []struct {
		name string
		read func(*storedfields.Reader)
	}{
		{name: "Float", read: func(r *storedfields.Reader) { r.Float("latitude") }},
		{name: "Fixed", read: func(r *storedfields.Reader) { r.Fixed("url hash", 12) }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			reader := storedfields.ReaderOf(stored.Record()[:4], errMalformed)
			testCase.read(reader)

			if err := reader.Err(); !errors.Is(err, errMalformed) {
				t.Errorf("Err = %v, want errMalformed", err)
			}
		})
	}
}

func TestAFieldLongerThanTheRecordIsMalformed(t *testing.T) {
	reader := storedfields.ReaderOf([]byte{0x40, 'a', 'b'}, errMalformed)
	reader.Text("title")

	if err := reader.Err(); !errors.Is(err, errMalformed) {
		t.Errorf("Err = %v, want errMalformed", err)
	}
}

func TestTheFirstMalformedFieldIsTheReportedOne(t *testing.T) {
	reader := storedfields.ReaderOf([]byte{0x40, 'a', 'b'}, errMalformed)
	reader.Text("title")
	first := reader.Err()
	reader.Text("author")

	if !errors.Is(reader.Err(), first) {
		t.Errorf("Err = %v, want the first failure %v", reader.Err(), first)
	}
}

func TestAFieldTheCallerRejectsIsMalformed(t *testing.T) {
	var stored storedfields.Writer
	stored.Text("xx")

	reader := storedfields.ReaderOf(stored.Record(), errMalformed)
	reader.Reject("language", errors.New("no such language"))

	if err := reader.Err(); !errors.Is(err, errMalformed) {
		t.Errorf("Err = %v, want errMalformed", err)
	}
}

func TestAFieldThatRunsOffTheEndOfTheRecordIsMalformed(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		record []byte
		read   func(*storedfields.Reader)
	}{
		{
			name:   "Varint",
			record: []byte{0x80},
			read:   func(r *storedfields.Reader) { r.Count("word count") },
		},
		{
			name:   "Text",
			record: []byte{0x80},
			read:   func(r *storedfields.Reader) { r.Text("title") },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			reader := storedfields.ReaderOf(testCase.record, errMalformed)
			testCase.read(reader)

			if err := reader.Err(); !errors.Is(err, errMalformed) {
				t.Errorf("Err = %v, want errMalformed", err)
			}
		})
	}
}

func TestBytesLeftCountsTheRecordNotYetRead(t *testing.T) {
	var stored storedfields.Writer
	stored.Byte(1)
	stored.Byte(2)

	reader := storedfields.ReaderOf(stored.Record(), errMalformed)
	if left := reader.BytesLeft(); left != 2 {
		t.Errorf("BytesLeft = %d, want 2", left)
	}
	reader.Byte("first")
	if left := reader.BytesLeft(); left != 1 {
		t.Errorf("BytesLeft after one field = %d, want 1", left)
	}
	reader.Reject("first", errors.New("out of range"))
	if left := reader.BytesLeft(); left != 0 {
		t.Errorf("BytesLeft after a malformed field = %d, want 0", left)
	}
}
