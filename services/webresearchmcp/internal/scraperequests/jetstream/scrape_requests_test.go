package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
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
		Name:      scraperequestcontract.ScrapeRequestsStreamName,
		Subjects:  []string{scraperequestcontract.ScrapeRequestSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}

	requests, err := jetstream.OpenScrapeRequests(
		serverURL,
		scraperequestcontract.ScrapeRequestSubject,
	)
	if err != nil {
		t.Fatalf("open the scrape requests: %v", err)
	}
	defer requests.Close()

	if err := requests.AskToScrape(
		ctx,
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	); err != nil {
		t.Fatalf("ask to scrape %s: %v", pageAddress, err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(
		ctx,
		scraperequestcontract.ScrapeRequestsStreamName,
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
	request, err := scraperequestcontract.UnmarshalScrapeRequest(message.Data())
	if err != nil {
		t.Fatalf("read the contract request: %v", err)
	}
	if request.PageURL.String() != pageAddress {
		t.Errorf("scrape request page url = %q, want %q", request.PageURL, pageAddress)
	}
}

func TestAskingToScrapeFailsWhileTheStreamIsMissing(t *testing.T) {
	serverURL := natstestserver.Start(t)

	requests, err := jetstream.OpenScrapeRequests(
		serverURL,
		scraperequestcontract.ScrapeRequestSubject,
	)
	if err != nil {
		t.Fatalf("open the scrape requests: %v", err)
	}
	defer requests.Close()

	if err := requests.AskToScrape(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	); err == nil {
		t.Fatal("asking to scrape answered without an error, want the missing stream to fail")
	}
}

func TestScrapeRequestsCannotOpenOnAServerThatIsAway(t *testing.T) {
	if _, err := jetstream.OpenScrapeRequests(
		"nats://127.0.0.1:1",
		scraperequestcontract.ScrapeRequestSubject,
	); err == nil {
		t.Fatal("the scrape requests opened, want the server that is away to fail")
	}
}
