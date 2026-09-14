package jetstream_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	scrapeschedules "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeschedules/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const pageURL = "https://example.org/a"

func scrapeRequestOf(t *testing.T) pagescrapecontract.ScrapeRequest {
	t.Helper()
	pageCanonicalURL := canonicalurltest.CanonicalURLOf(t, pageURL)
	return pagescrapecontract.ScrapeRequest{PageURL: pageCanonicalURL, FetchURL: pageCanonicalURL}
}

func TestDeferredScrapeComesBackOnTheRequestSubject(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := broker.CreateStream(ctx, jetstream.StreamConfig{
		Name: pagescrapecontract.ScrapeRequestsStreamName,
		Subjects: []string{
			pagescrapecontract.ScrapeRequestSubject,
			pagescrapecontract.EveryScrapeScheduleSubject,
		},
		AllowMsgSchedules: true,
	})
	if err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "pagescrapetest",
		FilterSubject: pagescrapecontract.ScrapeRequestSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create a scrape request consumer: %v", err)
	}
	request := scrapeRequestOf(t)

	if err := scrapeschedules.NewScrapeSchedules(broker, wallclock.Clock{}).
		ScheduleScrape(ctx, request, time.Second); err != nil {
		t.Fatalf("schedule the scrape: %v", err)
	}

	message, err := consumer.Next(jetstream.FetchMaxWait(30 * time.Second))
	if err != nil {
		t.Fatalf("take the redelivered request: %v", err)
	}
	redelivered, err := pagescrapecontract.UnmarshalScrapeRequest(message.Data())
	if err != nil {
		t.Fatalf("unmarshal the redelivered request: %v", err)
	}
	if redelivered.PageURL != request.PageURL {
		t.Errorf("redelivered the page %s, want %s", redelivered.PageURL, request.PageURL)
	}
}

func TestScrapeIsNotScheduledWithoutAStreamThatTakesSchedules(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))

	if err := scrapeschedules.NewScrapeSchedules(broker, wallclock.Clock{}).
		ScheduleScrape(ctx, scrapeRequestOf(t), time.Second); err == nil {
		t.Error("want an error scheduling a scrape no stream takes")
	}
}
