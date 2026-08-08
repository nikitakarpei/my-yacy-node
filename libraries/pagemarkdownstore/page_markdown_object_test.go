package pagemarkdownstore_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

func TestObjectNameIsDeterministicPerURL(t *testing.T) {
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
			first := pagemarkdownstore.ObjectName(tc.url)
			second := pagemarkdownstore.ObjectName(tc.url)
			if first != second {
				t.Fatalf("ObjectName not deterministic: %q vs %q", first, second)
			}
			if len(first) != 64 {
				t.Fatalf("ObjectName length = %d, want 64", len(first))
			}
		})
	}
}

func TestObjectNameDistinguishesURLs(t *testing.T) {
	if pagemarkdownstore.ObjectName("https://example.test/a") ==
		pagemarkdownstore.ObjectName("https://example.test/b") {
		t.Fatal("distinct URLs collided to one object name")
	}
}
