package crawlcapability_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type identityDerivation struct {
	name    string
	accepts crawlcapability.PageContentFormat
}

func (d identityDerivation) Name() string { return d.name }

func (d identityDerivation) Accepts(format crawlcapability.PageContentFormat) bool {
	return format == d.accepts
}

func (identityDerivation) Derive(
	page crawlcapability.CrawledPage,
	_ *crawlcapability.RenderedContent,
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

func TestBindRepresentationCarriesNameAndAccepts(t *testing.T) {
	output := crawlcapability.BindRepresentation(
		identityDerivation{name: "text", accepts: crawlcapability.PageContentFormatHTML},
		&recordingPublication{},
	)
	if output.Name() != "text" {
		t.Fatalf("name = %q, want text", output.Name())
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
		identityDerivation{name: "text", accepts: crawlcapability.PageContentFormatHTML},
		publication,
	)
	page := crawlcapability.CrawledPage{CanonicalURL: "http://example.com/a"}
	rendered := crawlcapability.NewRenderedContent(page.Body, page.Format)

	send, err := output.Prepare(page, rendered)
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
		failingDerivation{},
		&recordingPublication{},
	)
	rendered := crawlcapability.NewRenderedContent(nil, crawlcapability.PageContentFormatHTML)
	if _, err := output.Prepare(crawlcapability.CrawledPage{}, rendered); err == nil {
		t.Fatal("expected error when the derivation fails")
	}
}

type failingDerivation struct{}

func (failingDerivation) Name() string { return "failing" }

func (failingDerivation) Accepts(crawlcapability.PageContentFormat) bool { return true }

func (failingDerivation) Derive(
	crawlcapability.CrawledPage,
	*crawlcapability.RenderedContent,
) (crawlcapability.CrawledPage, error) {
	return crawlcapability.CrawledPage{}, errors.New("derive failed")
}
