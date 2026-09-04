package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
)

func crawledPagesStream(t *testing.T) natsjetstream.JetStream {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateOrUpdateStream(context.Background(), natsjetstream.StreamConfig{
		Name:     yacycrawlcontract.CrawledPagesStreamName,
		Subjects: []string{yacycrawlcontract.EveryCrawledPageSubject},
	}); err != nil {
		t.Fatal(err)
	}
	return js
}

func reportedPageOn(
	t *testing.T,
	js natsjetstream.JetStream,
	subject string,
) yacycrawlcontract.CrawledPage {
	t.Helper()
	consumer, err := js.CreateOrUpdateConsumer(
		context.Background(),
		yacycrawlcontract.CrawledPagesStreamName,
		natsjetstream.ConsumerConfig{
			FilterSubject: subject,
			AckPolicy:     natsjetstream.AckExplicitPolicy,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(natsjetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume %s: %v", subject, err)
	}
	_ = msg.Ack()
	page, err := yacycrawlcontract.UnmarshalCrawledPage(msg.Data())
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return page
}

func TestAnIndexablePageIsReportedOnTheIndexableSubject(t *testing.T) {
	js := crawledPagesStream(t)
	observer := &recordingCrawledPageReportObserver{}
	publisher := jetstream.New(js, observer)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")

	publisher.ReportIndexablePage(context.Background(), pageURL)

	page := reportedPageOn(t, js, yacycrawlcontract.IndexablePageSubject)
	if page.PageURL != pageURL {
		t.Fatalf("page url = %q, want %q", page.PageURL, pageURL)
	}
	if observer.reported != 1 {
		t.Fatalf("reported pages = %d, want 1", observer.reported)
	}
	if observer.indexing != jetstream.PageAllowsIndexing {
		t.Fatalf("indexing = %q, want %q", observer.indexing, jetstream.PageAllowsIndexing)
	}
}

func TestAPageThatRefusesIndexingIsReportedOnItsOwnSubject(t *testing.T) {
	js := crawledPagesStream(t)
	observer := &recordingCrawledPageReportObserver{}
	publisher := jetstream.New(js, observer)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://example.com/b")

	publisher.ReportIndexingRefusedPage(context.Background(), pageURL)

	page := reportedPageOn(t, js, yacycrawlcontract.IndexingRefusedPageSubject)
	if page.PageURL != pageURL {
		t.Fatalf("page url = %q, want %q", page.PageURL, pageURL)
	}
	if observer.indexing != jetstream.PageRefusesIndexing {
		t.Fatalf("indexing = %q, want %q", observer.indexing, jetstream.PageRefusesIndexing)
	}
}

func TestAReportThatNeverLeavesIsObserved(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	observer := &recordingCrawledPageReportObserver{}
	publisher := jetstream.New(js, observer)

	publisher.ReportIndexablePage(
		context.Background(), canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)

	if observer.reportingFailures != 1 {
		t.Fatalf("reporting failures = %d, want 1", observer.reportingFailures)
	}
	if observer.reported != 0 {
		t.Fatalf("reported pages = %d, want 0", observer.reported)
	}
}

type recordingCrawledPageReportObserver struct {
	reported          int
	encodingFailures  int
	reportingFailures int
	indexing          jetstream.PageIndexing
}

func (o *recordingCrawledPageReportObserver) CrawledPageReported(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing jetstream.PageIndexing,
) {
	o.reported++
	o.indexing = indexing
}

func (o *recordingCrawledPageReportObserver) CrawledPageEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing jetstream.PageIndexing,
	_ error,
) {
	o.encodingFailures++
	o.indexing = indexing
}

func (o *recordingCrawledPageReportObserver) CrawledPageReportingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing jetstream.PageIndexing,
	_ error,
) {
	o.reportingFailures++
	o.indexing = indexing
}
