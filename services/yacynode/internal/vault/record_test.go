package vault

import (
	"bytes"
	"errors"
	"testing"
)

func TestPayloadSurvivesTheRoundTrip(t *testing.T) {
	for _, payload := range [][]byte{{}, []byte("alpha")} {
		record := recordFrom(payload)
		if len(record) != len(payload)+1 {
			t.Fatalf("record of %q is %d bytes, want %d", payload, len(record), len(payload)+1)
		}

		got, err := payloadOf(record)
		if err != nil {
			t.Fatalf("payloadOf(%q): %v", record, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("payloadOf = %q, want %q", got, payload)
		}
	}
}

func TestUnreadableRecordIsRefused(t *testing.T) {
	for name, record := range map[string][]byte{
		"empty":            {},
		"truncated length": {0x80},
		"metadata present": append([]byte{3, 0x08, 0x01, 0x7f}, "alpha"...),
		"length overflow": {
			0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01,
		},
	} {
		if _, err := payloadOf(record); !errors.Is(err, errBadRecord) {
			t.Fatalf("payloadOf(%s) = %v, want errBadRecord", name, err)
		}
	}
}
