package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scraperequests/jetstream"
)

const (
	pageAddress   = "https://example.org/page"
	readMaxWait   = 5 * time.Second
	testDurable   = "webresearchmcp-test"
	askedPageURLs = 1
)

func TestAskingToScrapeWritesTheContractRequestOnTheStream(t *testing.T) {
	serverURL := natstestserver.Start(t)
	stream := natstestserver.ConnectJetStream(t, serverURL)
	ctx := context.Background()
	if _, err := stream.CreateOrUpdateStream(ctx, natsjetstream.StreamConfig{
		Name:      pagescrapecontract.ScrapeRequestsStreamName,
		Subjects:  []string{pagescrapecontract.ScrapeRequestSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}

	requests, err := jetstream.OpenScrapeRequests(
		serverURL, &recordingScrapeRequestPublicationObserver{},
	)
	if err != nil {
		t.Fatalf("open the scrape requests: %v", err)
	}
	defer requests.Close()

	requests.AskToScrape(ctx, canonicalurltest.CanonicalURLOf(t, pageAddress))

	consumer, err := stream.CreateOrUpdateConsumer(
		ctx,
		pagescrapecontract.ScrapeRequestsStreamName,
		natsjetstream.ConsumerConfig{
			Durable:   testDurable,
			AckPolicy: natsjetstream.AckExplicitPolicy,
		},
	)
	if err != nil {
		t.Fatalf("create the test consumer: %v", err)
	}
	message, err := consumer.Next(natsjetstream.FetchMaxWait(readMaxWait))
	if err != nil {
		t.Fatalf("read the scrape request: %v", err)
	}
	request, err := pagescrapecontract.UnmarshalScrapeRequest(message.Data())
	if err != nil {
		t.Fatalf("read the contract request: %v", err)
	}
	if request.PageURL.String() != pageAddress {
		t.Errorf("scrape request page url = %q, want %q", request.PageURL, pageAddress)
	}
	if !request.GivesUpOnDeferral {
		t.Error("the scrape request waits for a deferred origin, want it to give up")
	}
}

func TestARequestThatNeverLeavesIsObserved(t *testing.T) {
	serverURL := natstestserver.Start(t)
	observer := &recordingScrapeRequestPublicationObserver{}

	requests, err := jetstream.OpenScrapeRequests(serverURL, observer)
	if err != nil {
		t.Fatalf("open the scrape requests: %v", err)
	}
	defer requests.Close()

	requests.AskToScrape(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	)

	if observer.publishingFailures != 1 {
		t.Errorf("publishing failures = %d, want 1", observer.publishingFailures)
	}
	if observer.published != 0 {
		t.Errorf("published requests = %d, want none", observer.published)
	}
}

func TestScrapeRequestsCannotOpenOnAServerThatIsAway(t *testing.T) {
	if _, err := jetstream.OpenScrapeRequests(
		"nats://127.0.0.1:1", &recordingScrapeRequestPublicationObserver{},
	); err == nil {
		t.Fatal("the scrape requests opened, want the server that is away to fail")
	}
}

type recordingScrapeRequestPublicationObserver struct {
	published          int
	encodingFailures   int
	publishingFailures int
}

func (o *recordingScrapeRequestPublicationObserver) ScrapeRequestPublished(
	context.Context, canonicalurl.CanonicalURL,
) {
	o.published++
}

func (o *recordingScrapeRequestPublicationObserver) ScrapeRequestEncodingFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.encodingFailures++
}

func (o *recordingScrapeRequestPublicationObserver) ScrapeRequestPublishingFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.publishingFailures++
}
