package jetstream_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	scrapestreams "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapestreams/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestScrapeRequestsStreamTakesRequestsAndSchedules(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))

	stream, err := scrapestreams.CreateScrapeRequestsStream(
		ctx, broker, scrapestreams.ScrapeRequestsStreamLimits{MaxMsgs: 16},
	)
	if err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if !info.Config.AllowMsgSchedules {
		t.Error("the stream takes no scheduled message, so a deferred scrape has nowhere to wait")
	}
	if _, err := broker.Publish(
		ctx, pagescrapecontract.ScrapeRequestSubject, []byte("{}"),
	); err != nil {
		t.Errorf("publish a scrape request: %v", err)
	}
}

func TestScrapePageOffersStreamKeepsAPageOnlyWhileACorpusHasToReadIt(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))

	if err := scrapestreams.CreateScrapePageOffersStream(
		ctx, broker, scrapestreams.ScrapePageOffersStreamLimits{
			MaxBytes: 1 << 20,
			MaxAge:   time.Hour,
		},
	); err != nil {
		t.Fatalf("create the scrape page offers stream: %v", err)
	}

	stream, err := broker.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
	if err != nil {
		t.Fatalf("open the scrape page offers stream: %v", err)
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
