package pagemarkdownstore_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

func TestObjectNameOfIsDeterministicPerURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{name: "root", url: "https://example.test/"},
		{name: "path", url: "https://example.test/page"},
		{name: "query", url: "https://example.test/page?a=1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			canonicalURL := canonicalurltest.CanonicalURLOf(t, tc.url)
			first := pagemarkdownstore.ObjectNameOf(canonicalURL)
			second := pagemarkdownstore.ObjectNameOf(canonicalURL)
			if first != second {
				t.Fatalf("ObjectNameOf not deterministic: %q vs %q", first, second)
			}
			if len(first) != 64 {
				t.Fatalf("ObjectNameOf length = %d, want 64", len(first))
			}
		})
	}
}

func TestObjectNameOfDistinguishesURLs(t *testing.T) {
	if pagemarkdownstore.ObjectNameOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.test/a"),
	) ==
		pagemarkdownstore.ObjectNameOf(
			canonicalurltest.CanonicalURLOf(t, "https://example.test/b"),
		) {
		t.Fatal("distinct URLs collided to one object name")
	}
}
