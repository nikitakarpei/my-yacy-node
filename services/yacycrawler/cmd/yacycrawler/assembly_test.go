package main

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

func publishedRWIStreamOnly() []PageStreamConfig {
	representation := yacycrawlcontract.PageRepresentationKindRWI

	return []PageStreamConfig{{
		Representation: representation,
		Subject:        yacycrawlcontract.CrawledPageSubject(representation),
		MaxMsgs:        DefaultMaxMsgs,
		Published:      true,
	}}
}

func TestBuildPageRepresentationsSelectsTheConfiguredRepresentations(t *testing.T) {
	representations := buildPageRepresentations(
		nil,
		ServiceConfig{PageStreams: publishedRWIStreamOnly()},
	)
	if len(representations) != 1 ||
		representations[0].Kind() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("representations = %v, want the rwi representation alone", representations)
	}
}

func TestCatalogsDeriveConfiguredRepresentations(t *testing.T) {
	representations := buildPageRepresentations(
		nil,
		ServiceConfig{PageStreams: publishedRWIStreamOnly()},
	)
	admitted, err := admittedMediaTypesFor(ServiceConfig{MaxBodyBytes: 1 << 20})
	if err != nil {
		t.Fatalf("admit media types: %v", err)
	}
	if err := contentformatgraph.New(pageDerivationCatalog()).EnsureNoDanglingFormat(
		admitted.emittedFormats,
		representationContentFormats(representations),
	); err != nil {
		t.Fatalf("configured representation content is reachable, got %v", err)
	}
}

func TestBuildExtractorDefaultRegistersAll(t *testing.T) {
	admitted, err := admittedMediaTypesFor(ServiceConfig{MaxBodyBytes: 1 << 20})
	if err != nil {
		t.Fatalf("admit media types: %v", err)
	}
	extractor, err := buildExtractor(admitted)
	if err != nil {
		t.Fatalf("build extractor: %v", err)
	}
	if extractor == nil {
		t.Fatal("nil extractor")
	}
	// text/html routes to the html extractor, which yields the whole document.
	documents, err := extractor.ExtractDocuments(t.Context(), "http://h/p", "text/html",
		[]byte("<html><body><p>hello</p></body></html>"))
	if err != nil {
		t.Fatalf("dispatch to html extractor failed: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("want one extracted document, got %d", len(documents))
	}
}

func TestBuildExtractorAllowlistRestricts(t *testing.T) {
	admitted, err := admittedMediaTypesFor(ServiceConfig{
		MaxBodyBytes: 1 << 20, ContentTypes: []string{"text/html"},
	})
	if err != nil {
		t.Fatalf("admit media types: %v", err)
	}
	extractor, err := buildExtractor(admitted)
	if err != nil {
		t.Fatalf("build extractor: %v", err)
	}
	if _, err := extractor.ExtractDocuments(
		t.Context(),
		"http://h/a.zip",
		"application/zip",
		[]byte("x"),
	); err == nil {
		t.Fatal("zip should be unsupported when allowlist excludes it")
	}
}

func TestTraversalConfigMapsCaps(t *testing.T) {
	cfg := traversalConfig(ServiceConfig{RunPageBudget: 7, FrontierCap: 9})
	if cfg.RunPageBudget != 7 || cfg.MaxAdmittedURLs != 9 {
		t.Fatalf("traversal config not mapped: %+v", cfg)
	}
	if cfg.Frontier.MaxAttemptsPerURL != fetchRetryLimit ||
		cfg.Frontier.MaxDeferralsPerURL != maxDeferPerURL {
		t.Fatal("traversal config constants not applied")
	}
}
