package searchendpoint

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestSiteHashFromRequestHash(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{SiteHash: "ABCDEF"})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	got, ok := criteria.SiteHash.Get()
	if !ok || got.String() != "ABCDEF" {
		t.Fatalf("SiteHash = %q (present %v), want ABCDEF", got.String(), ok)
	}
}

func TestSiteHashFromOperatorBeforeStructuredHost(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{
		Modifier: "site:example.com",
		SiteHost: "ignored.example",
	})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}

	want, err := yacymodel.HashHost("example.com")
	if err != nil {
		t.Fatalf("HashHost: %v", err)
	}
	got, ok := criteria.SiteHash.Get()
	if !ok || got != want {
		t.Fatalf("SiteHash = %q (present %v), want %q", got.String(), ok, want.String())
	}
}

func TestSiteHashFromStructuredHostFallback(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{SiteHost: "example.com"})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}

	want, err := yacymodel.HashHost("example.com")
	if err != nil {
		t.Fatalf("HashHost: %v", err)
	}
	got, ok := criteria.SiteHash.Get()
	if !ok || got != want {
		t.Fatalf("SiteHash = %q (present %v), want %q", got.String(), ok, want.String())
	}
}

func TestLanguageFromOperatorBeforeStructured(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{
		Modifier: "/language/de",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	got, ok := criteria.Language.Get()
	if !ok || got.String() != "de" {
		t.Fatalf("Language = %q (present %v), want de", got.String(), ok)
	}
}

func TestStructuredLanguageDoesNotFilter(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{Language: "en"})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	if criteria.Language.Present() {
		t.Fatalf("Language = %v, want absent", criteria.Language)
	}
}

func TestWrongLengthLanguageOperatorDoesNotFilter(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{Modifier: "/language/deu"})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	if criteria.Language.Present() {
		t.Fatalf("Language = %v, want absent", criteria.Language)
	}
}

func TestInvalidSiteHashIsRejected(t *testing.T) {
	if _, err := criteriaFromRequest(yacyproto.SearchRequest{SiteHash: "!!"}); err == nil {
		t.Fatal("criteriaFromRequest accepted an invalid site hash")
	}
}

func TestInvalidSiteHostIsRejected(t *testing.T) {
	if _, err := criteriaFromRequest(yacyproto.SearchRequest{SiteHost: "."}); err == nil {
		t.Fatal("criteriaFromRequest accepted an invalid site host")
	}
}

func TestContentKindFollowsContentDomain(t *testing.T) {
	cases := []struct {
		domain yacyproto.SearchContentDomain
		kind   searchcriteria.ContentKind
	}{
		{yacyproto.ContentDomainImage, searchcriteria.ImageContent},
		{yacyproto.ContentDomainAudio, searchcriteria.AudioContent},
		{yacyproto.ContentDomainVideo, searchcriteria.VideoContent},
		{yacyproto.ContentDomainApp, searchcriteria.ApplicationContent},
		{yacyproto.ContentDomainText, searchcriteria.AnyContent},
	}
	for _, c := range cases {
		t.Run(string(c.domain), func(t *testing.T) {
			criteria, err := criteriaFromRequest(yacyproto.SearchRequest{ContentDom: c.domain})
			if err != nil {
				t.Fatalf("criteriaFromRequest: %v", err)
			}
			if criteria.ContentKind != c.kind {
				t.Errorf("ContentKind = %v, want %v", criteria.ContentKind, c.kind)
			}
		})
	}
}

func TestMissingCountAndTimeTakeDefaults(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	if criteria.MaxResults != defaultSearchCount {
		t.Errorf("MaxResults = %d, want %d", criteria.MaxResults, defaultSearchCount)
	}
	if criteria.TimeLimit != defaultSearchTime {
		t.Errorf("TimeLimit = %v, want %v", criteria.TimeLimit, defaultSearchTime)
	}
}

func TestTimeBeyondTheMaximumIsClamped(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{
		Time: 10_000,
	})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	if criteria.TimeLimit != maxSearchTime {
		t.Errorf("TimeLimit = %v, want %v", criteria.TimeLimit, maxSearchTime)
	}
}

func TestTimeBelowTheMaximumIsKept(t *testing.T) {
	criteria, err := criteriaFromRequest(yacyproto.SearchRequest{
		Time: 500,
	})
	if err != nil {
		t.Fatalf("criteriaFromRequest: %v", err)
	}
	if criteria.TimeLimit != 500*time.Millisecond {
		t.Errorf("TimeLimit = %v, want %v", criteria.TimeLimit, 500*time.Millisecond)
	}
}
