package main

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type receiveSignal struct {
	urlmeta.URLReceiver
	received chan struct{}
}

func (s receiveSignal) Receive(
	ctx context.Context,
	metadata []yacymodel.URLMetadata,
) (urlmeta.Receipt, error) {
	receipt, err := s.URLReceiver.Receive(ctx, metadata)
	close(s.received)
	return receipt, err
}

type discardedHolds struct{}

func (discardedHolds) ObserveHeld(int)     {}
func (discardedHolds) ObserveReleased(int) {}

func TestCrawlRuntimeConsumesIngestBatch(t *testing.T) {
	storage, err := openNodeStorage(openTestVault(t), discardedHolds{})
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	received := make(chan struct{})
	storage.urlReceiver = receiveSignal{URLReceiver: storage.urlReceiver, received: received}

	cfg := crawlConfig{
		NATSURL:       natstestserver.Start(t),
		IngestSubject: defaultIngestSubject,
		IngestDurable: defaultIngestDurable,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	createCrawlerStreams(t, ctx, cfg)

	runtime, err := buildCrawlRuntime(ctx, cfg, storage)
	if err != nil {
		t.Fatalf("build crawl runtime: %v", err)
	}

	done := make(chan struct{})
	go func() { runtime.Run(ctx); close(done) }()

	publishIngestChunk(t, ctx, cfg, yacycrawlcontract.PageRWIMetadataChunk{
		CanonicalURL: "https://example.org",
		Metadata:     []yacymodel.URLMetadata{{Address: "https://example.org"}},
	})

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("ingest batch was not received")
	}

	count, err := storage.urlDirectory.Count(ctx)
	if err != nil {
		t.Fatalf("count urls: %v", err)
	}
	if count != 1 {
		t.Fatalf("url count = %d, want 1", count)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not stop after cancel")
	}
	runtime.Close()
}

func createCrawlerStreams(t *testing.T, ctx context.Context, cfg crawlConfig) {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, cfg.NATSURL)
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx, js, yacycrawlcontract.PageRepresentationKindRWI,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: cfg.IngestSubject, MaxMsgs: 64},
	); err != nil {
		t.Fatalf("create ingest stream: %v", err)
	}
}

func publishIngestChunk(
	t *testing.T,
	ctx context.Context,
	cfg crawlConfig,
	chunk yacycrawlcontract.PageRWIMetadataChunk,
) {
	t.Helper()
	data, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal ingest chunk: %v", err)
	}
	js := natstestserver.ConnectJetStream(t, cfg.NATSURL)
	if _, err := js.Publish(ctx, cfg.IngestSubject, data); err != nil {
		t.Fatalf("publish ingest chunk: %v", err)
	}
}
