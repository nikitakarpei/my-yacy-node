package pagelinks_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagelinks"
)

func canonicalURLFor(t *testing.T, address string) canonicalurl.CanonicalURL {
	t.Helper()
	canonical, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func TestTheBaseOfLinksIsTheBaseHRefResolvedAgainstThePage(t *testing.T) {
	page := canonicalURLFor(t, "https://site.example/dir/page")
	for baseHRef, wantedBase := range map[string]string{
		"/other/":      "https://site.example/other/",
		"":             "https://site.example/dir/page",
		"mailto:a@b.c": "https://site.example/dir/page",
	} {
		if base := pagelinks.BaseFrom(page, baseHRef).String(); base != wantedBase {
			t.Fatalf("base from %q is %q, want %q", baseHRef, base, wantedBase)
		}
	}
}

func TestWebLinksAreTheDistinctCanonicalWebAddresses(t *testing.T) {
	base := canonicalURLFor(t, "https://site.example/dir/page")

	links := pagelinks.WebLinksFrom([]string{
		"a.html", " /b#part ", "mailto:a@b.c", "javascript:void(0)",
		"HTTPS://Other.example:443/c", "a.html#again", "http://[broken",
	}, base)

	wantedLinks := []string{
		"https://site.example/dir/a.html",
		"https://site.example/b",
		"https://other.example/c",
	}
	if len(links) != len(wantedLinks) {
		t.Fatalf("links %v, want %v", links, wantedLinks)
	}
	for position, link := range links {
		if link.String() != wantedLinks[position] {
			t.Fatalf("links %v, want %v", links, wantedLinks)
		}
	}
}

func TestDataLinksAreWrittenOrEncodedWebAddresses(t *testing.T) {
	base := canonicalURLFor(t, "https://site.example/dir/page")

	links := pagelinks.DataLinksFrom([]string{
		"aHR0cHM6Ly9zcGFtLmV4YW1wbGUvb2ZmZXI=", " /local ", "https://other.example/x",
		"plain words", "aHR0cGdhcmJhZ2U=", "aHR0c!!!", "http://[broken",
	}, base)

	wantedLinks := []struct {
		address string
		encoded bool
	}{{"https://spam.example/offer", true}, {"https://site.example/local", false}, {"https://other.example/x", false}}
	if len(links) != len(wantedLinks) {
		t.Fatalf("links %v, want %v", links, wantedLinks)
	}
	for position, link := range links {
		if link.Address.String() != wantedLinks[position].address ||
			link.Encoded != wantedLinks[position].encoded {
			t.Fatalf("links %v, want %v", links, wantedLinks)
		}
	}
}
