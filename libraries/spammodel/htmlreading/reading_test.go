package htmlreading_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagelinks"
)

const readablePage = `<html><head><title> Cheap
  pills </title><style>p { color: red }</style><base href="/first/"><base href="/second/"></head>
<body><p>Buy <b>now</b></p><script>track()</script><svg><title>icon</title><text>drawn</text></svg>after
<a href="offer.html">offer</a><a href="">empty</a><a>none</a><a xlink:href="namespaced">x</a>
<div data-target="https://spam.example/" data-empty="">block</div>
<span data-link="aHR0cHM6Ly9lbmNvZGVkLmV4YW1wbGUv"></span></body></html>`

func canonicalURLFor(t *testing.T, address string) canonicalurl.CanonicalURL {
	t.Helper()
	canonical, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func readingFrom(t *testing.T, contentType, body string) htmlreading.Reading {
	t.Helper()
	return htmlreading.ReadingFrom(
		canonicalURLFor(t, "https://site.example/dir/page"), []byte(body), contentType,
	)
}

func TestAReadingKeepsTheVisibleTitleAndText(t *testing.T) {
	reading := readingFrom(t, "text/html", readablePage)

	if reading.Title != "Cheap pills" ||
		reading.VisibleText != "Buy now after offer empty none x block" {
		t.Fatalf("title %q and visible text %q", reading.Title, reading.VisibleText)
	}
}

func TestAReadingResolvesTheWrittenLinksAgainstTheFirstBase(t *testing.T) {
	reading := readingFrom(t, "text/html", readablePage)

	wantedLinks := []canonicalurl.CanonicalURL{
		canonicalURLFor(t, "https://site.example/first/offer.html"),
	}
	wantedDataLinks := []pagelinks.DataLink{
		{Address: canonicalURLFor(t, "https://spam.example/")},
		{Address: canonicalURLFor(t, "https://encoded.example/"), Encoded: true},
	}
	if !slices.Equal(reading.WebLinks, wantedLinks) ||
		!slices.Equal(reading.DataLinks, wantedDataLinks) {
		t.Fatalf("web links %v, data links %v", reading.WebLinks, reading.DataLinks)
	}
}

func TestAReadingKeepsItsAddressAndBodySize(t *testing.T) {
	reading := readingFrom(t, "text/html", readablePage)

	if reading.Address != canonicalURLFor(t, "https://site.example/dir/page") ||
		reading.BodyBytes != len(readablePage) {
		t.Fatalf("address %v, body bytes %d", reading.Address, reading.BodyBytes)
	}
}

func TestAReadingDecodesTheBodyByItsCharset(t *testing.T) {
	for name, page := range map[string]struct {
		contentType string
		body        string
		wantedText  string
	}{
		"declared in the content type":  {"text/html; charset=windows-1251", "<p>\xcf\xf0\xe8</p>", "При"},
		"declared in the markup":        {"text/html", "<meta charset=\"iso-8859-1\"><p>caf\xe9</p>", "café"},
		"utf-8 without a declaration":   {"text/html", "<p>café</p>", "café"},
		"guessed without a declaration": {"text/html", "<p>\x93quoted\x94</p>", "“quoted”"},
	} {
		t.Run(name, func(t *testing.T) {
			reading := readingFrom(t, page.contentType, page.body)

			if reading.VisibleText != page.wantedText {
				t.Fatalf("visible text %q, want %q", reading.VisibleText, page.wantedText)
			}
		})
	}
}

func TestAReadingKeepsTheDecodedHTML(t *testing.T) {
	reading := readingFrom(t, "text/html; charset=windows-1251", "<p>\xcf\xf0\xe8</p>")

	if reading.HTML != "<p>При</p>" {
		t.Fatalf("html %q", reading.HTML)
	}
}
