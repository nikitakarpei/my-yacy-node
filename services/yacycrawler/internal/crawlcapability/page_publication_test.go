package crawlcapability_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
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

func TestBindRepresentationCarriesRepresentationAndAccepts(t *testing.T) {
	output := crawlcapability.BindRepresentation(
		yacycrawlcontract.PageRepresentationText,
		identityDerivation{accepts: crawlcapability.PageContentFormatHTML},
		&recordingPublication{},
	)
	if output.Representation() != yacycrawlcontract.PageRepresentationText {
		t.Fatalf("representation = %q", output.Representation())
	}
	if !output.Accepts(crawlcapability.PageContentFormatHTML) {
		t.Fatal("should accept html")
	}
	if output.Accepts(crawlcapability.PageContentFormatText) {
		t.Fatal("should not accept text")
	}
}

func TestPrepareDerivesOnceAndPublishOnEachCallResends(t *testing.T) {
	publication := &recordingPublication{}
	output := crawlcapability.BindRepresentation(
		yacycrawlcontract.PageRepresentationText,
		identityDerivation{accepts: crawlcapability.PageContentFormatHTML},
		publication,
	)
	page := crawlcapability.CrawledPage{CanonicalURL: "http://example.com/a"}
	rendered := crawlcapability.NewRenderedContent(page.Body, page.Format)

	send, err := output.Prepare(page, rendered.In)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := send(context.Background()); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := send(context.Background()); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(publication.published) != 2 {
		t.Fatalf(
			"want 2 publish calls from retrying the same send, got %d",
			len(publication.published),
		)
	}
}

func TestPrepareFailsWhenDerivationFails(t *testing.T) {
	output := crawlcapability.BindRepresentation(
		yacycrawlcontract.PageRepresentationText,
		failingDerivation{},
		&recordingPublication{},
	)
	rendered := crawlcapability.NewRenderedContent(nil, crawlcapability.PageContentFormatHTML)
	if _, err := output.Prepare(crawlcapability.CrawledPage{}, rendered.In); err == nil {
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
