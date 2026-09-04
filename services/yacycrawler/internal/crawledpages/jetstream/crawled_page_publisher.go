// Package jetstream publishes every page the crawler read, on the subject the page's own
// indexing statement selects. A page that never leaves goes to the observer; the caller
// learns nothing and visits the next page.
package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
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
	pageURL canonicalurl.CanonicalURL,
) {
	p.publish(ctx, pageURL, indexablePagePublication)
}

func (p *CrawledPagePublisher) PublishIndexingRefusedPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	p.publish(ctx, pageURL, indexingRefusedPagePublication)
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
	pageURL canonicalurl.CanonicalURL,
	publication crawledPagePublication,
) {
	data, err := yacycrawlcontract.MarshalCrawledPage(
		yacycrawlcontract.CrawledPage{PageURL: pageURL},
	)
	if err != nil {
		p.observer.CrawledPageEncodingFailed(ctx, pageURL, publication.indexing, err)
		return
	}
	if _, err := p.stream.Publish(ctx, publication.subject, data); err != nil {
		p.observer.CrawledPagePublishingFailed(ctx, pageURL, publication.indexing, err)
		return
	}
	p.observer.CrawledPagePublished(ctx, pageURL, publication.indexing)
}
