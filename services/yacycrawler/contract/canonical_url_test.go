package yacycrawlcontract_test

import (
	"encoding/json"
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
		if got.String() != want {
			t.Errorf("CanonicalURLOf(%q) = %q, want %q", input, got.String(), want)
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

func TestCanonicalURLOfLink(t *testing.T) {
	base, err := yacycrawlcontract.CanonicalURLOf("http://example.com/dir/page")
	if err != nil {
		t.Fatalf("CanonicalURLOf: %v", err)
	}
	got, err := base.CanonicalURLOfLink("../other")
	if err != nil {
		t.Fatalf("CanonicalURLOfLink: %v", err)
	}
	if got.String() != "http://example.com/other" {
		t.Fatalf("got %q", got.String())
	}
}

func TestCanonicalURLOfLinkRejectsBadInput(t *testing.T) {
	base, err := yacycrawlcontract.CanonicalURLOf("http://h/")
	if err != nil {
		t.Fatalf("CanonicalURLOf: %v", err)
	}
	if _, err := base.CanonicalURLOfLink("::bad"); err == nil {
		t.Error("bad link should error")
	}
	if _, err := (yacycrawlcontract.CanonicalURL{}).CanonicalURLOfLink("relative"); err == nil {
		t.Error("relative link on an unset base should error")
	}
}

func TestCanonicalURLRoundTripsThroughJSON(t *testing.T) {
	canonical, err := yacycrawlcontract.CanonicalURLOf("HTTP://Example.COM/a")
	if err != nil {
		t.Fatalf("CanonicalURLOf: %v", err)
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"http://example.com/a"` {
		t.Fatalf("marshalled to %s", data)
	}
	var decoded yacycrawlcontract.CanonicalURL
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != canonical {
		t.Fatalf("decoded %q, want %q", decoded.String(), canonical.String())
	}
}

func TestCanonicalURLRejectsUncanonicalJSON(t *testing.T) {
	for _, data := range []string{`"HTTP://Example.COM/a"`, `"http://example.com/a/../b"`, `"ftp://h/"`, `5`} {
		var decoded yacycrawlcontract.CanonicalURL
		if err := json.Unmarshal([]byte(data), &decoded); err == nil {
			t.Errorf("unmarshal %s should error", data)
		}
	}
}

func TestCanonicalURLHostnameExcludesThePort(t *testing.T) {
	for input, want := range map[string]string{
		"HTTP://Example.COM/a":      "example.com",
		"http://example.com:8080/x": "example.com",
	} {
		got, err := yacycrawlcontract.CanonicalURLOf(input)
		if err != nil {
			t.Fatalf("CanonicalURLOf(%q): %v", input, err)
		}
		if got.Hostname() != want {
			t.Errorf("CanonicalURLOf(%q).Hostname() = %q, want %q", input, got.Hostname(), want)
		}
	}
}

func TestCanonicalURLHasQuery(t *testing.T) {
	for input, want := range map[string]bool{
		"http://example.com/a":     false,
		"http://example.com/a?q=1": true,
	} {
		got, err := yacycrawlcontract.CanonicalURLOf(input)
		if err != nil {
			t.Fatalf("CanonicalURLOf(%q): %v", input, err)
		}
		if got.HasQuery() != want {
			t.Errorf("CanonicalURLOf(%q).HasQuery() = %v, want %v", input, got.HasQuery(), want)
		}
	}
}

func TestCanonicalURLDecodedFromJSONCarriesItsParts(t *testing.T) {
	var decoded yacycrawlcontract.CanonicalURL
	if err := json.Unmarshal([]byte(`"http://example.com/a?q=1"`), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Hostname() != "example.com" || !decoded.HasQuery() {
		t.Fatalf("hostname = %q, hasQuery = %v", decoded.Hostname(), decoded.HasQuery())
	}
}
