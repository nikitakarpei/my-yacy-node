package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCanonicalURLOf(t *testing.T) {
	cases := map[string]string{
		"HTTP://Example.COM":         "http://example.com/",
		"http://example.com:80/a":    "http://example.com/a",
		"https://example.com:443/a":  "https://example.com/a",
		"http://example.com/a/../b":  "http://example.com/b",
		"http://example.com/a/#frag": "http://example.com/a/",
		"http://example.com:8080/x":  "http://example.com:8080/x",
		"http://example.com/a/b/":    "http://example.com/a/b/",
	}
	for input, want := range cases {
		got, err := yacycrawlcontract.CanonicalURLOf(input)
		if err != nil {
			t.Fatalf("CanonicalURLOf(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("CanonicalURLOf(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCanonicalURLOfRejectsBadInput(t *testing.T) {
	for _, input := range []string{"::bad", "ftp://example.com/x", "http:///path"} {
		if _, err := yacycrawlcontract.CanonicalURLOf(input); err == nil {
			t.Errorf("CanonicalURLOf(%q) should error", input)
		}
	}
}

func TestCanonicalURLOfReference(t *testing.T) {
	got, err := yacycrawlcontract.CanonicalURLOfReference("http://example.com/dir/page", "../other")
	if err != nil {
		t.Fatalf("CanonicalURLOfReference: %v", err)
	}
	if got != "http://example.com/other" {
		t.Fatalf("got %q", got)
	}
}

func TestCanonicalURLOfReferenceRejectsBadInput(t *testing.T) {
	if _, err := yacycrawlcontract.CanonicalURLOfReference("::bad", "x"); err == nil {
		t.Error("bad base should error")
	}
	if _, err := yacycrawlcontract.CanonicalURLOfReference("http://h/", "::bad"); err == nil {
		t.Error("bad ref should error")
	}
}
