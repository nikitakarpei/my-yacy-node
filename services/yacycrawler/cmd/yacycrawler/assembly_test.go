package main

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type unrenderableFeed struct{}

func (unrenderableFeed) Representation() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindText
}

func (unrenderableFeed) ContentFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatHTML
}

func (unrenderableFeed) Derive(
	crawlcapability.CrawledPage,
	[]byte,
) (crawlcapability.PagePublication, error) {
	return crawlcapability.PagePublication{}, nil
}

func (unrenderableFeed) Publish(context.Context, crawlcapability.PagePublication) error {
	return nil
}

func TestBuildPageFeedsSelectsTheConfiguredRepresentations(t *testing.T) {
	feeds := buildPageFeeds(nil, ServiceConfig{PageFeeds: []PageFeedConfig{rwiOutputConfig()}})
	if len(feeds) != 1 || feeds[0].Representation() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("feeds = %v, want the rwi feed alone", feeds)
	}
}

func TestBuildPageRenderingsCoversWhatTheFeedsRead(t *testing.T) {
	feeds := buildPageFeeds(nil, ServiceConfig{PageFeeds: []PageFeedConfig{rwiOutputConfig()}})
	renderings, err := buildPageRenderings(feeds)
	if err != nil {
		t.Fatalf("build page renderings: %v", err)
	}
	if len(renderings) != 1 || renderings[0].Format() != crawlcapability.PageContentFormatText {
		t.Fatalf("renderings = %v, want text alone", renderings)
	}
}

func TestBuildPageRenderingsRejectsUnrenderableContentFormat(t *testing.T) {
	_, err := buildPageRenderings([]crawlcapability.PageFeed{unrenderableFeed{}})
	if err == nil {
		t.Fatal("feed reading a format no rendering produces should error")
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
	// text/html routes to the html extractor.
	if _, err := extractor.Extract(t.Context(), "http://h/p", "text/html",
		[]byte("<html><body></body></html>")); err == nil {
		t.Fatal("expected unextractable for empty html, dispatch reached extractor")
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
