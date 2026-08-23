package pagelinks_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagelinks"
)

func TestLinksResolveAgainstTheBaseHref(t *testing.T) {
	page := `<!DOCTYPE html><html lang="en"><head><title>T</title>
<base href="http://other.example/dir/"></head>
<body><a href="page">relative</a></body></html>`

	links, err := pagelinks.PageLinksFrom(
		t.Context(), "http://host.example/p", "text/html", []byte(page),
	)
	if err != nil {
		t.Fatalf("PageLinksFrom: %v", err)
	}
	if len(links.LinkedURLs) != 1 ||
		links.LinkedURLs[0].String() != "http://other.example/dir/page" {
		t.Fatalf("linked = %v", links.LinkedURLs)
	}
	if links.LocalLinks != 1 || links.ExternalLinks != 0 {
		t.Fatalf("links local=%d external=%d", links.LocalLinks, links.ExternalLinks)
	}
}

func TestOnlyAbsoluteLinksSurviveAPageWithoutACanonicalURL(t *testing.T) {
	page := `<!DOCTYPE html><html lang="en"><head><title>T</title></head>
<body><a href="/relative">relative</a>
<a href="http://other.example/ext">absolute</a></body></html>`

	links, err := pagelinks.PageLinksFrom(t.Context(), "file:///tmp/p", "text/html", []byte(page))
	if err != nil {
		t.Fatalf("PageLinksFrom: %v", err)
	}
	if len(links.LinkedURLs) != 1 ||
		links.LinkedURLs[0].String() != "http://other.example/ext" {
		t.Fatalf("linked = %v", links.LinkedURLs)
	}
}

func TestARepeatedLinkCountsOnce(t *testing.T) {
	page := `<html><body><a href="/a">one</a><a href="/a">again</a>
<a href="http://other.example/x">out</a></body></html>`

	links, err := pagelinks.PageLinksFrom(
		t.Context(), "http://host.example/p", "text/html; charset=utf-8", []byte(page),
	)
	if err != nil {
		t.Fatalf("PageLinksFrom: %v", err)
	}
	if len(links.LinkedURLs) != 2 || links.LocalLinks != 1 || links.ExternalLinks != 1 {
		t.Fatalf("linked=%v local=%d external=%d",
			links.LinkedURLs, links.LocalLinks, links.ExternalLinks)
	}
}

func TestMetaRobotsRefusals(t *testing.T) {
	for content, want := range map[string][2]bool{
		"noindex":           {true, false},
		"nofollow":          {false, true},
		"none":              {true, true},
		"noindex, nofollow": {true, true},
		"index, follow":     {false, false},
	} {
		page := `<html><head><meta name="robots" content="` + content +
			`"></head><body></body></html>`
		links, err := pagelinks.PageLinksFrom(
			t.Context(), "http://host.example/p", "text/html", []byte(page),
		)
		if err != nil {
			t.Fatalf("PageLinksFrom %q: %v", content, err)
		}
		if links.RefusesIndexing != want[0] || links.RefusesLinkDiscovery != want[1] {
			t.Errorf("%q: indexing=%v discovery=%v",
				content, links.RefusesIndexing, links.RefusesLinkDiscovery)
		}
	}
}

func TestABodyThatIsNotHTMLHoldsNoLinks(t *testing.T) {
	_, err := pagelinks.PageLinksFrom(
		t.Context(), "http://host.example/p", "application/pdf", []byte("%PDF-1.4"),
	)
	if !errors.Is(err, pagelinks.ErrNotHTML) {
		t.Fatalf("err = %v, want ErrNotHTML", err)
	}
}

func TestAnUnparsableContentTypeFallsBackToItsLeadingSegment(t *testing.T) {
	page := `<html><body><a href="http://other.example/x">out</a></body></html>`

	links, err := pagelinks.PageLinksFrom(
		t.Context(), "http://host.example/p", "text/html;;charset", []byte(page),
	)
	if err != nil {
		t.Fatalf("PageLinksFrom: %v", err)
	}
	if len(links.LinkedURLs) != 1 {
		t.Fatalf("linked = %v", links.LinkedURLs)
	}
}
