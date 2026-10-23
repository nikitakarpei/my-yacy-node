// Package jetstream publishes every page the crawler read, on the subject the page's own
// indexing statement selects. A page that does not leave goes to the observer, and the
// caller learns that it did not leave.
package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type CrawledPagePublisher struct {
	stream   jetstream.JetStream
	observer CrawledPagePublicationObserver
}

func New(
	stream jetstream.JetStream,
	observer CrawledPagePublicationObserver,
) *CrawledPagePublisher {
	return &CrawledPagePublisher{stream: stream, observer: observer}
}

func (p *CrawledPagePublisher) PublishIndexablePage(
	ctx context.Context,
	page yacycrawlcontract.CrawledPage,
) bool {
	return p.publish(ctx, page, indexablePagePublication)
}

func (p *CrawledPagePublisher) PublishIndexingRefusedPage(
	ctx context.Context,
	page yacycrawlcontract.CrawledPage,
) bool {
	return p.publish(ctx, page, indexingRefusedPagePublication)
}

type crawledPagePublication struct {
	subject  string
	indexing PageIndexing
}

var (
	indexablePagePublication = crawledPagePublication{
		subject:  yacycrawlcontract.IndexablePageSubject,
		indexing: PageAllowsIndexing,
	}
	indexingRefusedPagePublication = crawledPagePublication{
		subject:  yacycrawlcontract.IndexingRefusedPageSubject,
		indexing: PageRefusesIndexing,
	}
)

func (p *CrawledPagePublisher) publish(
	ctx context.Context,
	page yacycrawlcontract.CrawledPage,
	publication crawledPagePublication,
) bool {
	data, err := yacycrawlcontract.MarshalCrawledPage(page)
	if err != nil {
		p.observer.CrawledPageEncodingFailed(ctx, page.PageURL, publication.indexing, err)
		return false
	}
	if _, err := p.stream.Publish(ctx, publication.subject, data); err != nil {
		p.observer.CrawledPagePublishingFailed(ctx, page.PageURL, publication.indexing, err)
		return false
	}
	p.observer.CrawledPagePublished(ctx, page.PageURL, publication.indexing)
	return true
}
