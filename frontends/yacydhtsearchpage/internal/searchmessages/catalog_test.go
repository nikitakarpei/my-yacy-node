package searchmessages_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchmessages"
)

func TestTextOfFillsThePlaceholders(t *testing.T) {
	text, err := searchmessages.English().TextOf("results.none", "yacy")
	if err != nil {
		t.Fatalf("TextOf: %v", err)
	}
	if text != "No results for “yacy”." {
		t.Fatalf("TextOf = %q", text)
	}
}

func TestTextOfRefusesAnUnknownKey(t *testing.T) {
	_, err := searchmessages.English().TextOf("no.such.key")
	if err == nil {
		t.Fatalf("TextOf of an unknown key = %v, want an error", err)
	}
}
