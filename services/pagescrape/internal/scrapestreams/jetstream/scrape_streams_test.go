package jetstream_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	scrapestreams "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapestreams/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestScrapeRequestsStreamKeepsARequestOnlyUntilItIsFinished(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := scrapestreams.CreateScrapeRequestsStream(
		ctx, broker, scrapestreams.ScrapeRequestsStreamLimits{MaxMsgs: 16},
	)
	if err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}
	if _, err := broker.Publish(
		ctx, pagescrapecontract.ScrapeRequestSubject, []byte("{}"),
	); err != nil {
		t.Fatalf("publish a scrape request: %v", err)
	}
	consumer, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "scraper",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create a scraper consumer: %v", err)
	}
	request, err := consumer.Next()
	if err != nil {
		t.Fatalf("take the request: %v", err)
	}
	if err := request.DoubleAck(ctx); err != nil {
		t.Fatalf("finish the request: %v", err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if info.State.Msgs != 0 {
		t.Errorf("the stream kept %d finished requests, want none", info.State.Msgs)
	}
}

func TestScrapeRequestsStreamRejectsARequestWhileFull(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := scrapestreams.CreateScrapeRequestsStream(
		ctx, broker, scrapestreams.ScrapeRequestsStreamLimits{MaxMsgs: 1},
	); err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}
	if _, err := broker.Publish(
		ctx, pagescrapecontract.ScrapeRequestSubject, []byte("{}"),
	); err != nil {
		t.Fatalf("publish the first request: %v", err)
	}

	if _, err := broker.Publish(
		ctx, pagescrapecontract.ScrapeRequestSubject, []byte("{}"),
	); err == nil {
		t.Error("the full stream took a second request")
	}
}

func TestScrapePageOffersStreamKeepsAPageOnlyWhileACorpusHasToReadIt(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))

	stream, err := scrapestreams.CreateScrapePageOffersStream(
		ctx, broker, scrapestreams.ScrapePageOffersStreamLimits{MaxBytes: 1 << 20},
	)
	if err != nil {
		t.Fatalf("create the scrape page offers stream: %v", err)
	}
	if _, err := broker.Publish(
		ctx, pagescrapecontract.OfferedPageSubject, []byte("{}"),
	); err != nil {
		t.Fatalf("offer a page: %v", err)
	}
	info, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if info.State.Msgs != 0 {
		t.Errorf("the stream kept %d offers no corpus reads, want none", info.State.Msgs)
	}
}

func TestScrapePageOffersStreamRejectsAnOfferWhileFull(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	offer := make([]byte, 600)
	stream, err := scrapestreams.CreateScrapePageOffersStream(
		ctx, broker, scrapestreams.ScrapePageOffersStreamLimits{MaxBytes: int64(len(offer)) + 200},
	)
	if err != nil {
		t.Fatalf("create the scrape page offers stream: %v", err)
	}
	if _, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "corpus",
		AckPolicy: jetstream.AckExplicitPolicy,
	}); err != nil {
		t.Fatalf("create a corpus consumer: %v", err)
	}
	if _, err := broker.Publish(ctx, pagescrapecontract.OfferedPageSubject, offer); err != nil {
		t.Fatalf("offer the first page: %v", err)
	}

	if _, err := broker.Publish(ctx, pagescrapecontract.OfferedPageSubject, offer); err == nil {
		t.Error("the full stream took another offer")
	}
	info, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if info.State.FirstSeq != 1 {
		t.Errorf("first kept offer = %d, want the unread first offer kept", info.State.FirstSeq)
	}
}
