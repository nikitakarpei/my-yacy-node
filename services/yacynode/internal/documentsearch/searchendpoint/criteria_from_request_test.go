package searchendpoint

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
