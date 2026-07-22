package main

import (
	"context"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type underivableFeed struct{}

func (underivableFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindText
}

func (underivableFeed) ContentFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormat("unproduced")
}

func (underivableFeed) Derive(
	crawlcapability.CrawledPage,
	[]byte,
) (crawlcapability.PagePublication, error) {
	return crawlcapability.PagePublication{}, nil
}

func (underivableFeed) Publish(context.Context, crawlcapability.PagePublication) error {
	return nil
}

type markdownContentFeed struct{ underivableFeed }

func (markdownContentFeed) ContentFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatMarkdown
}

func TestBuildPageFeedsSelectsTheConfiguredRepresentations(t *testing.T) {
	feeds := buildPageFeeds(nil, ServiceConfig{PageStreams: publishedPageStreams()})
	if len(feeds) != 1 || feeds[0].Representation() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("feeds = %v, want the rwi feed alone", feeds)
	}
}

func TestBuildPageDerivationsGivesRWIOnlyTheFullTextDerivation(t *testing.T) {
	feeds := buildPageFeeds(nil, ServiceConfig{PageStreams: publishedPageStreams()})
	derivations, err := buildPageDerivations(feeds)
	if err != nil {
		t.Fatalf("build page derivations: %v", err)
	}
	produced := make([]crawlcapability.PageContentFormat, 0, len(derivations))
	for _, derivation := range derivations {
		produced = append(produced, derivation.TargetFormat())
	}
	want := []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatFullText}
	if !slices.Equal(produced, want) {
		t.Fatalf("rwi reads full-text, want %v, got %v", want, produced)
	}
}

func TestBuildPageDerivationsResolvesTheChainToWhatAFeedReads(t *testing.T) {
	derivations, err := buildPageDerivations([]crawlcapability.PageFeed{markdownContentFeed{}})
	if err != nil {
		t.Fatalf("build page derivations: %v", err)
	}
	produced := make([]crawlcapability.PageContentFormat, 0, len(derivations))
	for _, derivation := range derivations {
		produced = append(produced, derivation.TargetFormat())
	}
	want := []crawlcapability.PageContentFormat{
		crawlcapability.PageContentFormatReadableHTML,
		crawlcapability.PageContentFormatMarkdown,
	}
	if !slices.Equal(produced, want) {
		t.Fatalf("chain = %v, want dependency order %v", produced, want)
	}
}

func TestBuildPageDerivationsRejectsUnderivableContentFormat(t *testing.T) {
	_, err := buildPageDerivations([]crawlcapability.PageFeed{underivableFeed{}})
	if err == nil {
		t.Fatal("feed reading a format no derivation produces should error")
	}
}

func TestBuildExtractorDefaultRegistersAll(t *testing.T) {
	extractor, err := buildExtractor(ServiceConfig{MaxBodyBytes: 1 << 20})
	if err != nil {
		t.Fatalf("build extractor: %v", err)
	}
	if extractor == nil {
		t.Fatal("nil extractor")
	}
	// text/html routes to the html extractor, which yields the whole document.
	documents, err := extractor.Extract(t.Context(), "http://h/p", "text/html",
		[]byte("<html><body><p>hello</p></body></html>"))
	if err != nil {
		t.Fatalf("dispatch to html extractor failed: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("want one extracted document, got %d", len(documents))
	}
}

func TestBuildExtractorAllowlistRestricts(t *testing.T) {
	extractor, err := buildExtractor(ServiceConfig{
		MaxBodyBytes: 1 << 20, ContentTypes: []string{"text/html"},
	})
	if err != nil {
		t.Fatalf("build extractor: %v", err)
	}
	if _, err := extractor.Extract(
		t.Context(),
		"http://h/a.zip",
		"application/zip",
		[]byte("x"),
	); err == nil {
		t.Fatal("zip should be unsupported when allowlist excludes it")
	}
}

func TestBuildExtractorEmptyActiveSetErrors(t *testing.T) {
	if _, err := buildExtractor(ServiceConfig{
		MaxBodyBytes: 1 << 20, ContentTypes: []string{"application/unregistered"},
	}); err == nil {
		t.Fatal("allowlist matching nothing should error")
	}
}

func TestAllowedMediaTypes(t *testing.T) {
	if allowedMediaTypes(nil) != nil {
		t.Fatal("empty list should yield nil (all allowed)")
	}
	allow := allowedMediaTypes([]string{"text/html"})
	if !allow["text/html"] || allow["application/zip"] {
		t.Fatalf("unexpected allow set: %v", allow)
	}
}

func TestTraversalConfigMapsCaps(t *testing.T) {
	cfg := traversalConfig(ServiceConfig{RunPageBudget: 7, FrontierCap: 9})
	if cfg.RunPageBudget != 7 || cfg.FrontierCapacity != 9 {
		t.Fatalf("traversal config not mapped: %+v", cfg)
	}
	if cfg.FetchRetryLimit != fetchRetryLimit || cfg.MaxDeferralsPerURL != maxDeferPerURL {
		t.Fatal("traversal config constants not applied")
	}
}
