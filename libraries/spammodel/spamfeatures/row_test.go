package spamfeatures_test

import (
	"math"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/featurehashing"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagelinks"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

func canonicalURLFor(t *testing.T, address string) canonicalurl.CanonicalURL {
	t.Helper()
	canonical, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func canonicalURLsFor(t *testing.T, addresses ...string) []canonicalurl.CanonicalURL {
	t.Helper()
	canonicalURLs := make([]canonicalurl.CanonicalURL, 0, len(addresses))
	for _, address := range addresses {
		canonicalURLs = append(canonicalURLs, canonicalURLFor(t, address))
	}
	return canonicalURLs
}

func assertFamily(
	t *testing.T, row spamfeatures.Row, family spamfeatures.Family, wantedFeatures []string,
) {
	t.Helper()
	if wanted := featurehashing.EntriesFrom(
		wantedFeatures,
	); !slices.Equal(
		row.HashedEntries[family],
		wanted,
	) {
		t.Fatalf(
			"%s entries %v, want the entries of %q",
			family,
			row.HashedEntries[family],
			wantedFeatures,
		)
	}
}

func TestTheTextFamilyReadsTheTitleAndTheStartOfTheVisibleText(t *testing.T) {
	visibleText := "Buy " + strings.Repeat("é", 2496) + " unread"

	row := spamfeatures.RowFrom(htmlreading.Reading{
		Address:     canonicalURLFor(t, "https://site.example/"),
		Title:       "Cheap",
		VisibleText: visibleText,
	}, nil)

	assertFamily(t, row, spamfeatures.Text, featurehashing.CharacterNgramsFrom(
		"Cheap "+visibleText[:len("Buy ")+2496*len("é")],
		featurehashing.NgramSizes{Shortest: 4, Longest: 4},
	))
}

func TestTheAddressFamilyReadsTheWordsOfTheHostPathAndQuery(t *testing.T) {
	row := spamfeatures.RowFrom(htmlreading.Reading{
		Address: canonicalURLFor(t, "https://Best-Pills.example/buy/Now.html?id=7&x=%C3%A9"),
	}, nil)

	assertFamily(t, row, spamfeatures.Address, featurehashing.CharacterNgramsFrom(
		"best pills example buy now html id 7 x c3 a9",
		featurehashing.NgramSizes{Shortest: 3, Longest: 5},
	))
}

func TestTheMarkupFamilyReadsTagsAndAttributeMarkersOfTheStartOfTheMarkup(t *testing.T) {
	markup := `<HTML><Div style="display: none" data-x="1">` + strings.Repeat(" ", 30000) + "<p>"

	row := spamfeatures.RowFrom(htmlreading.Reading{
		Address: canonicalURLFor(t, "https://site.example/"),
		HTML:    markup,
	}, nil)

	assertFamily(t, row, spamfeatures.Markup, featurehashing.WordNgramsFrom(
		"html div attr_style attr_data-x",
		featurehashing.NgramSizes{Shortest: 1, Longest: 2},
	))
}

func TestTheLinksFamilyReadsOutboundDomainsAndLocalRedirectPaths(t *testing.T) {
	row := spamfeatures.RowFrom(htmlreading.Reading{
		Address: canonicalURLFor(t, "https://www.site.example/page"),
		WebLinks: canonicalURLsFor(t,
			"https://site.example/about",
			"https://site.example/Go/casino",
			"https://shop.spam.example/Offer/deal?x=1",
		),
		DataLinks: []pagelinks.DataLink{
			{Address: canonicalURLFor(t, "https://spam.example/"), Encoded: true},
		},
	}, nil)

	assertFamily(t, row, spamfeatures.Links, []string{
		"localpath:go",
		"domain:spam.example", "host:shop.spam.example", "domainpath:spam.example/offer",
		"domain:spam.example", "host:spam.example", "domainpath:spam.example/",
	})
}

func TestTheHeadersFamilyReadsHeaderNamesProxiesKeepProductsAndGenerators(t *testing.T) {
	row := spamfeatures.RowFrom(htmlreading.Reading{
		Address: canonicalURLFor(t, "https://site.example/"),
		HTML:    `<meta name="generator" content="WordPress 6.1">`,
	}, http.Header{
		"Age":              {"3"},
		"Cache-Status":     {"squid;hit"},
		"Connection":       {"keep-alive"},
		"Set-Cookie":       {"session=1"},
		"Content-Encoding": {"gzip"},
		"Server":           {"Apache/2.4 (Debian)"},
		"X-Powered-By":     {" "},
		"Content-Type":     {"text/html"},
	})

	assertFamily(t, row, spamfeatures.Headers, []string{
		"name:server", "server=apache",
		"name:x-powered-by", "x-powered-by=",
		"name:content-type",
		"generator=wordpress",
	})
}

func TestTheMetaValuesDescribeTheHostTextMarkupAndLinks(t *testing.T) {
	markup := `<a rel="sponsored" style="display:none"><a href="/x"><script></script>`
	row := spamfeatures.RowFrom(htmlreading.Reading{
		Address:     canonicalURLFor(t, "https://www.cheap-pills4u.example/a/b?c=1"),
		VisibleText: "Welcome to Cheap PILLS",
		HTML:        markup,
		BodyBytes:   999,
		WebLinks: canonicalURLsFor(t,
			"https://cheap-pills4u.example/go/x",
			"https://spam.example/?aff=1",
			"https://spam.example/b",
			"https://other.example/",
		),
		DataLinks: []pagelinks.DataLink{
			{Address: canonicalURLFor(t, "https://spam.example/c"), Encoded: true},
			{Address: canonicalURLFor(t, "https://cheap-pills4u.example/d")},
		},
	}, nil)

	wantedMeta := []float64{
		math.Log1p(22), math.Log1p(999), math.Log1p(1), math.Log1p(3), 3.0 / 5,
		1, 1, 25.0 / 20, 2, 1, 4.0 / 5, 1, 1.0 / 5, 2.0 / 100, 1.0 / 20,
		math.Log1p(1), math.Log1p(1), math.Log1p(1), math.Log1p(2), math.Log1p(1), 3.0 / 4,
	}
	if !slices.Equal(row.MetaValues, wantedMeta) {
		t.Fatalf("meta %v, want %v", row.MetaValues, wantedMeta)
	}
}

func TestTheHostNamingValueTellsWhetherTheTextNamesAHostWord(t *testing.T) {
	for address, wantedHostNaming := range map[string]float64{
		"https://www.pharmacy.example/": 0,
		"https://ab.cd/":                1,
	} {
		row := spamfeatures.RowFrom(htmlreading.Reading{
			Address:     canonicalURLFor(t, address),
			VisibleText: "nothing here",
		}, nil)

		if hostNaming := row.MetaValues[11]; hostNaming != wantedHostNaming {
			t.Fatalf("host naming of %s is %v, want %v", address, hostNaming, wantedHostNaming)
		}
	}
}

func TestAPageWithoutOutboundLinksHasNoTopOutboundDomainShare(t *testing.T) {
	row := spamfeatures.RowFrom(
		htmlreading.Reading{Address: canonicalURLFor(t, "https://site.example/")}, nil)

	if topOutboundDomainShare := row.MetaValues[20]; topOutboundDomainShare != 0 {
		t.Fatalf("top outbound domain share %v", topOutboundDomainShare)
	}
}

func TestTheWordsOfEachFamilyAreTheWordsTheRowHashes(t *testing.T) {
	page := htmlreading.Reading{
		Address:     canonicalURLFor(t, "https://site.example/buy"),
		Title:       "Cheap",
		VisibleText: "Buy cheap pills now",
		HTML:        `<div rel="sponsored"><a href="https://shop.example/go">shop</a></div>`,
		WebLinks: canonicalURLsFor(
			t,
			"https://shop.example/go",
			"https://site.example/out/1",
		),
	}
	responseHeaders := http.Header{"Server": {"nginx/1.2"}}

	words := spamfeatures.WordsFrom(page, responseHeaders)

	row := spamfeatures.RowFrom(page, responseHeaders)
	for _, family := range spamfeatures.Families {
		if len(words[family]) == 0 {
			t.Errorf("%s has no words", family)
		}
		assertFamily(t, row, family, words[family])
	}
}
