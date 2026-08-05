package documentsearch

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestSiteHashFromRequestHash(t *testing.T) {
	criteria, err := searchCriteriaFromRequest(yacyproto.SearchRequest{SiteHash: "ABCDEF"})
	if err != nil {
		t.Fatalf("searchCriteriaFromRequest: %v", err)
	}
	got, ok := criteria.siteHash.Get()
	if !ok || got.String() != "ABCDEF" {
		t.Fatalf("siteHash = %q (present %v), want ABCDEF", got.String(), ok)
	}
}

func TestSiteHashFromOperatorBeforeStructuredHost(t *testing.T) {
	criteria, err := searchCriteriaFromRequest(yacyproto.SearchRequest{
		Modifier: "site:example.com",
		SiteHost: "ignored.example",
	})
	if err != nil {
		t.Fatalf("searchCriteriaFromRequest: %v", err)
	}

	want, err := yacymodel.HashHost("example.com")
	if err != nil {
		t.Fatalf("HashHost: %v", err)
	}
	got, ok := criteria.siteHash.Get()
	if !ok || got != want {
		t.Fatalf("siteHash = %q (present %v), want %q", got.String(), ok, want.String())
	}
}

func TestSiteHashFromStructuredHostFallback(t *testing.T) {
	criteria, err := searchCriteriaFromRequest(yacyproto.SearchRequest{SiteHost: "example.com"})
	if err != nil {
		t.Fatalf("searchCriteriaFromRequest: %v", err)
	}

	want, err := yacymodel.HashHost("example.com")
	if err != nil {
		t.Fatalf("HashHost: %v", err)
	}
	got, ok := criteria.siteHash.Get()
	if !ok || got != want {
		t.Fatalf("siteHash = %q (present %v), want %q", got.String(), ok, want.String())
	}
}

func TestLanguageFromOperatorBeforeStructured(t *testing.T) {
	criteria, err := searchCriteriaFromRequest(yacyproto.SearchRequest{
		Modifier: "/language/de",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("searchCriteriaFromRequest: %v", err)
	}
	got, ok := criteria.language.Get()
	if !ok || got.String() != "de" {
		t.Fatalf("language = %q (present %v), want de", got.String(), ok)
	}
}

func TestStructuredLanguageDoesNotFilter(t *testing.T) {
	criteria, err := searchCriteriaFromRequest(yacyproto.SearchRequest{Language: "en"})
	if err != nil {
		t.Fatalf("searchCriteriaFromRequest: %v", err)
	}
	if criteria.language.Present() {
		t.Fatalf("language = %v, want absent", criteria.language)
	}
}

func TestSearchReportsRequestedTermsAlongsideWantedTerms(t *testing.T) {
	word, related := hashFor("w1"), hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word:    {postingEntry(word, "u1", 0, 1), postingEntry(word, "u2", 0, 1)},
		related: {postingEntry(related, "u2", 0, 1), postingEntry(related, "u3", 0, 1)},
	}}
	s := searcher{
		index:              index,
		documentDirectory:  fakeDirectory{documentDirectory: urlMetadata("u1", "u2")},
		maxPostingsPerTerm: 100,
	}

	result, err := s.search(
		context.Background(),
		searchCriteria{terms: []yacymodel.Hash{word}},
		requestedMatchReport{mode: reportRequestedTerms, terms: []yacymodel.Hash{related}},
	)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := result.documentsMatchingEachReportedTerm[related]; !hasExactlyDocuments(
		got,
		"u2",
		"u3",
	) {
		t.Fatalf("documentsMatchingEachReportedTerm[related] = %v, want u2, u3", got)
	}
}
