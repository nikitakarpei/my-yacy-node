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

func publishedPageOn(
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

func TestAnIndexablePageIsPublishedOnTheIndexableSubject(t *testing.T) {
	js := crawledPagesStream(t)
	observer := &recordingCrawledPagePublicationObserver{}
	publisher := jetstream.New(js, observer)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")

	publisher.PublishIndexablePage(context.Background(), pageURL)

	page := publishedPageOn(t, js, yacycrawlcontract.IndexablePageSubject)
	if page.PageURL != pageURL {
		t.Fatalf("page url = %q, want %q", page.PageURL, pageURL)
	}
	if observer.published != 1 {
		t.Fatalf("published pages = %d, want 1", observer.published)
	}
	if observer.indexing != jetstream.PageAllowsIndexing {
		t.Fatalf("indexing = %q, want %q", observer.indexing, jetstream.PageAllowsIndexing)
	}
}

func TestAPageThatRefusesIndexingIsPublishedOnItsOwnSubject(t *testing.T) {
	js := crawledPagesStream(t)
	observer := &recordingCrawledPagePublicationObserver{}
	publisher := jetstream.New(js, observer)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://example.com/b")

	publisher.PublishIndexingRefusedPage(context.Background(), pageURL)

	page := publishedPageOn(t, js, yacycrawlcontract.IndexingRefusedPageSubject)
	if page.PageURL != pageURL {
		t.Fatalf("page url = %q, want %q", page.PageURL, pageURL)
	}
	if observer.indexing != jetstream.PageRefusesIndexing {
		t.Fatalf("indexing = %q, want %q", observer.indexing, jetstream.PageRefusesIndexing)
	}
}

func TestAPageThatNeverLeavesIsObserved(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	observer := &recordingCrawledPagePublicationObserver{}
	publisher := jetstream.New(js, observer)

	publisher.PublishIndexablePage(
		context.Background(), canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)

	if observer.publishingFailures != 1 {
		t.Fatalf("publishing failures = %d, want 1", observer.publishingFailures)
	}
	if observer.published != 0 {
		t.Fatalf("published pages = %d, want 0", observer.published)
	}
}

type recordingCrawledPagePublicationObserver struct {
	published          int
	encodingFailures   int
	publishingFailures int
	indexing           jetstream.PageIndexing
}

func (o *recordingCrawledPagePublicationObserver) CrawledPagePublished(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing jetstream.PageIndexing,
) {
	o.published++
	o.indexing = indexing
}

func (o *recordingCrawledPagePublicationObserver) CrawledPageEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing jetstream.PageIndexing,
	_ error,
) {
	o.encodingFailures++
	o.indexing = indexing
}

func (o *recordingCrawledPagePublicationObserver) CrawledPagePublishingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing jetstream.PageIndexing,
	_ error,
) {
	o.publishingFailures++
	o.indexing = indexing
}
