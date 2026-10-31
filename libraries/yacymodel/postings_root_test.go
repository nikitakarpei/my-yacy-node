package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestRootBytesRoundTrip(t *testing.T) {
	postings := randomPagePostings(seededRandom(8), 5)
	root := rootOf(t, wholePagePostingsOf(t, postings, partitionsFrom(t, 9)))

	parsed, err := yacymodel.ParsePostingsRootBytes(root.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if parsed != root {
		t.Errorf("parsed root = %+v, want %+v", parsed, root)
	}
}

func TestParsePostingsRootBytesRejectsMalformedBytes(t *testing.T) {
	digest := make([]byte, 32)
	for name, raw := range map[string][]byte{
		"too short":             append([]byte{1}, digest...),
		"too long":              append([]byte{1, 4, 0}, digest...),
		"other format":          append([]byte{2, 4}, digest...),
		"exponent out of range": append([]byte{1, 63}, digest...),
	} {
		if _, err := yacymodel.ParsePostingsRootBytes(raw); err == nil {
			t.Errorf("%s: parse accepted %x", name, raw)
		}
	}
}
