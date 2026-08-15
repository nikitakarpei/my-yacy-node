package yacymodel_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestEncodeKnownVectors(t *testing.T) {
	cases := []struct {
		in   []byte
		want string
	}{
		{nil, ""},
		{[]byte{0}, "AA"},
		{[]byte{255}, "_w"},
		{[]byte{0, 0, 0}, "AAAA"},
	}
	for _, c := range cases {
		if got := yacymodel.Encode(c.in); got != c.want {
			t.Errorf("Encode(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	inputs := [][]byte{
		{},
		{1},
		{1, 2},
		{1, 2, 3},
		{0xde, 0xad, 0xbe, 0xef},
		bytes.Repeat([]byte{0xa5}, 16),
	}
	for _, in := range inputs {
		got, err := yacymodel.Decode(yacymodel.Encode(in))
		if err != nil {
			t.Fatalf("Decode round trip %v: %v", in, err)
		}
		if len(in) == 0 && len(got) == 0 {
			continue
		}
		if !bytes.Equal(got, in) {
			t.Errorf("round trip %v = %v", in, got)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	if _, err := yacymodel.Decode("AA=A"); !errors.Is(err, yacymodel.ErrInvalidBase64) {
		t.Fatalf("Decode invalid = %v, want ErrInvalidBase64", err)
	}
}

func TestDecodeIgnoresLineFeeds(t *testing.T) {
	got, err := yacymodel.Decode("AQ\nID")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("Decode with newline = %v", got)
	}
}
