package pagefeed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
)

type identityDerivation struct {
	accepts crawlcapability.PageContentFormat
}

func (d identityDerivation) Accepts(format crawlcapability.PageContentFormat) bool {
	return format == d.accepts
}

func (identityDerivation) Derive(
	page crawlcapability.CrawledPage,
	_ crawlcapability.RenderContent,
) (crawlcapability.CrawledPage, error) {
	return page, nil
}

type recordingPublication struct {
	published []crawlcapability.CrawledPage
	failWith  error
}

func (p *recordingPublication) Publish(_ context.Context, page crawlcapability.CrawledPage) error {
	if p.failWith != nil {
		return p.failWith
	}
	p.published = append(p.published, page)
	return nil
}

func TestBindCarriesRepresentationAndAccepts(t *testing.T) {
	feed := pagefeed.Bind(
		yacycrawlcontract.PageRepresentationKindText,
		identityDerivation{accepts: crawlcapability.PageContentFormatHTML},
		&recordingPublication{},
	)
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindText {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if !feed.Accepts(crawlcapability.PageContentFormatHTML) {
		t.Fatal("should accept html")
	}
	if feed.Accepts(crawlcapability.PageContentFormatText) {
		t.Fatal("should not accept text")
	}
}

func TestDeriveHappensOnceAndEachPublishResends(t *testing.T) {
	publication := &recordingPublication{}
	feed := pagefeed.Bind(
		yacycrawlcontract.PageRepresentationKindText,
		identityDerivation{accepts: crawlcapability.PageContentFormatHTML},
		publication,
	)
	page := crawlcapability.CrawledPage{CanonicalURL: "http://example.com/a"}
	rendered := renderPage(page)

	publish, err := feed.Derive(page, rendered)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if err := publish(context.Background()); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := publish(context.Background()); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(publication.published) != 2 {
		t.Fatalf(
			"want 2 publish calls from retrying the same publish, got %d",
			len(publication.published),
		)
	}
}

func TestDeriveFailsWhenTheDerivationFails(t *testing.T) {
	feed := pagefeed.Bind(
		yacycrawlcontract.PageRepresentationKindText,
		failingDerivation{},
		&recordingPublication{},
	)
	rendered := renderPage(
		crawlcapability.CrawledPage{Format: crawlcapability.PageContentFormatHTML},
	)
	if _, err := feed.Derive(crawlcapability.CrawledPage{}, rendered); err == nil {
		t.Fatal("expected error when the derivation fails")
	}
}

type failingDerivation struct{}

func (failingDerivation) Accepts(crawlcapability.PageContentFormat) bool { return true }

func (failingDerivation) Derive(
	crawlcapability.CrawledPage,
	crawlcapability.RenderContent,
) (crawlcapability.CrawledPage, error) {
	return crawlcapability.CrawledPage{}, errors.New("derive failed")
}

func renderPage(page crawlcapability.CrawledPage) crawlcapability.RenderContent {
	return func(rendering crawlcapability.ContentRendering) ([]byte, error) {
		return rendering.Render(page.Body, page.Format)
	}
}
