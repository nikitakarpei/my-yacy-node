package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/scraperequests/jetstream"
)

func TestPublishWritesTheContractMessageForACapturedPage(t *testing.T) {
	serverURL := natstestserver.Start(t)
	js := natstestserver.ConnectJetStream(t, serverURL)
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, natsjetstream.StreamConfig{
		Name:      pagescrapecontract.ScrapeRequestsStreamName,
		Subjects:  []string{pagescrapecontract.ScrapeRequestSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatalf("create scrape requests stream: %v", err)
	}

	publisher, err := jetstream.Open(serverURL)
	if err != nil {
		t.Fatalf("open publisher: %v", err)
	}
	defer publisher.Close()

	const pageURL = "https://example.com/"
	const replayURL = "http://pywb:8080/archive/20240101120000mp_/https://example.com/"
	if err := publisher.Publish(
		ctx,
		canonicalurltest.CanonicalURLOf(t, pageURL),
		canonicalurltest.CanonicalURLOf(t, replayURL),
	); err != nil {
		t.Fatalf("publish %s: %v", pageURL, err)
	}

	consumer, err := js.CreateOrUpdateConsumer(
		ctx,
		pagescrapecontract.ScrapeRequestsStreamName,
		natsjetstream.ConsumerConfig{AckPolicy: natsjetstream.AckExplicitPolicy},
	)
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	message, err := consumer.Next(natsjetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("read scrape request: %v", err)
	}
	request, err := pagescrapecontract.UnmarshalScrapeRequest(message.Data())
	if err != nil {
		t.Fatalf("unmarshal scrape request: %v", err)
	}
	if request.PageURL.String() != pageURL {
		t.Fatalf("scrape request page url = %q, want %q", request.PageURL, pageURL)
	}
	if request.FetchURL.String() != replayURL {
		t.Fatalf("scrape request fetch url = %q, want %q", request.FetchURL, replayURL)
	}
}

func TestPublishFailsWhenTheScrapeRequestsStreamIsMissing(t *testing.T) {
	serverURL := natstestserver.Start(t)

	publisher, err := jetstream.Open(serverURL)
	if err != nil {
		t.Fatalf("open publisher: %v", err)
	}
	defer publisher.Close()

	if err := publisher.Publish(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		canonicalurltest.CanonicalURLOf(
			t,
			"http://pywb:8080/archive/1mp_/https://example.com/",
		),
	); err == nil {
		t.Fatal("publish: want an error naming the missing stream")
	}
}

func TestOpenFailsWhenTheBrokerIsUnreachable(t *testing.T) {
	if _, err := jetstream.Open("nats://127.0.0.1:1"); err == nil {
		t.Fatal("open publisher: want an error")
	}
}
