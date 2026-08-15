package yacymodel_test

import (
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestURLNormalformOmitsTheDefaultPortAndLowercasesTheHost(t *testing.T) {
	cases := map[string]string{
		"http://Example.COM:80/Path": "http://example.com/Path",
		"https://Example.com:443/":   "https://example.com/",
		"http://example.com:8080/x":  "http://example.com:8080/x",
	}
	for address, want := range cases {
		if got := normalformOfAddress(t, address).String(); got != want {
			t.Errorf("normalform of %q = %q, want %q", address, got, want)
		}
	}
}

func TestURLNormalformDropsTheSessionID(t *testing.T) {
	cases := map[string]string{
		"http://example.com/a?sid=1&b=2":        "http://example.com/a?b=2",
		"http://example.com/a?b=2&sid=1":        "http://example.com/a?b=2",
		"http://example.com/a?sid=1":            "http://example.com/a",
		"http://example.com/a?jsessionid=1&b=2": "http://example.com/a?b=2",
	}
	for address, want := range cases {
		if got := normalformOfAddress(t, address).String(); got != want {
			t.Errorf("normalform of %q = %q, want %q", address, got, want)
		}
	}
}

func TestURLNormalformResolvesDotSegments(t *testing.T) {
	cases := map[string]string{
		"http://example.com/a/b/../c":    "http://example.com/a/c",
		"http://example.com/a/b/..":      "http://example.com/a",
		"http://example.com/a/./b":       "http://example.com/a/b",
		"http://example.com/a//b":        "http://example.com/a/b",
		"http://example.com/a/b/../../c": "http://example.com/c",
		"http://example.com/../a":        "http://example.com/a",
		"http://example.com/..":          "http://example.com/",
		"http://example.com":             "http://example.com/",
		"http://example.com/a/b/c":       "http://example.com/a/b/c",
	}
	for address, want := range cases {
		if got := normalformOfAddress(t, address).String(); got != want {
			t.Errorf("normalform of %q = %q, want %q", address, got, want)
		}
	}
}

func TestURLNormalformLeavesTheQueryUntouched(t *testing.T) {
	const address = "http://example.com/a/b/../c?x=/../y"
	want := "http://example.com/a/c?x=/../y"
	if got := normalformOfAddress(t, address).String(); got != want {
		t.Errorf("normalform of %q = %q, want %q", address, got, want)
	}
}

func TestURLNormalformEncodesNonBasicLabelsPerLabel(t *testing.T) {
	cases := map[string]string{
		"http://example.com/":    "http://example.com/",
		"http://bücher.example/": "http://xn--bcher-kva.example/",
		"http://münchen.de/":     "http://xn--mnchen-3ya.de/",
		"http://例え.テスト/":         "http://xn--r8jz45g.xn--zckzah/",
		"http://www.bücher.com/": "http://www.xn--bcher-kva.com/",
		"http://127.0.0.1/":      "http://127.0.0.1/",
	}
	for address, want := range cases {
		if got := normalformOfAddress(t, address).String(); got != want {
			t.Errorf("normalform of %q = %q, want %q", address, got, want)
		}
	}
}

func normalformOfAddress(t *testing.T, raw string) yacymodel.URLNormalform {
	t.Helper()

	address, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}

	return yacymodel.URLNormalformOf(address)
}
