package pagefeed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
)

type identityDerivation struct{}

func (identityDerivation) Derive(
	page crawlcapability.CrawledPage,
	_ []byte,
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

func TestBindCarriesRepresentationAndContentFormat(t *testing.T) {
	feed := pagefeed.Bind(
		yacycrawlcontract.PageRepresentationKindText,
		crawlcapability.PageContentFormatText,
		identityDerivation{},
		&recordingPublication{},
	)
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindText {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != crawlcapability.PageContentFormatText {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}

func TestDeriveHappensOnceAndEachPublishResends(t *testing.T) {
	publication := &recordingPublication{}
	feed := pagefeed.Bind(
		yacycrawlcontract.PageRepresentationKindText,
		crawlcapability.PageContentFormatText,
		identityDerivation{},
		publication,
	)
	page := crawlcapability.CrawledPage{CanonicalURL: "http://example.com/a"}

	publish, err := feed.Derive(page, []byte("hello"))
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
		crawlcapability.PageContentFormatText,
		failingDerivation{},
		&recordingPublication{},
	)
	if _, err := feed.Derive(crawlcapability.CrawledPage{}, nil); err == nil {
		t.Fatal("expected error when the derivation fails")
	}
}

type failingDerivation struct{}

func (failingDerivation) Derive(
	crawlcapability.CrawledPage,
	[]byte,
) (crawlcapability.CrawledPage, error) {
	return crawlcapability.CrawledPage{}, errors.New("derive failed")
}
