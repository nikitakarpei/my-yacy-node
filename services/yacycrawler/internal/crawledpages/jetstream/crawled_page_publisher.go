// Package jetstream reports every page the crawler read, on the subject the page's own
// indexing statement selects. A report that never leaves goes to the observer; the caller
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
	observer CrawledPageReportObserver
}

func New(
	stream jetstream.JetStream,
	observer CrawledPageReportObserver,
) *CrawledPagePublisher {
	return &CrawledPagePublisher{stream: stream, observer: observer}
}

func (p *CrawledPagePublisher) ReportIndexablePage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	p.report(ctx, pageURL, indexablePageReport)
}

func (p *CrawledPagePublisher) ReportIndexingRefusedPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	p.report(ctx, pageURL, indexingRefusedPageReport)
}

type crawledPageReport struct {
	subject  string
	indexing PageIndexing
}

var (
	indexablePageReport = crawledPageReport{
		subject:  yacycrawlcontract.IndexablePageSubject,
		indexing: PageAllowsIndexing,
	}
	indexingRefusedPageReport = crawledPageReport{
		subject:  yacycrawlcontract.IndexingRefusedPageSubject,
		indexing: PageRefusesIndexing,
	}
)

func (p *CrawledPagePublisher) report(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	report crawledPageReport,
) {
	data, err := yacycrawlcontract.MarshalCrawledPage(
		yacycrawlcontract.CrawledPage{PageURL: pageURL},
	)
	if err != nil {
		p.observer.CrawledPageEncodingFailed(ctx, pageURL, report.indexing, err)
		return
	}
	if _, err := p.stream.Publish(ctx, report.subject, data); err != nil {
		p.observer.CrawledPageReportingFailed(ctx, pageURL, report.indexing, err)
		return
	}
	p.observer.CrawledPageReported(ctx, pageURL, report.indexing)
}
