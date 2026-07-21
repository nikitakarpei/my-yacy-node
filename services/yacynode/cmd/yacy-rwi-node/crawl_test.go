package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawldispatch"
)

func TestCrawlRuntimeDispatchAndConsume(t *testing.T) {
	storage, err := openNodeStorage(openTestVault(t))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}

	cfg := crawlConfig{
		NATSURL:       natstestserver.Start(t),
		OrdersSubject: defaultOrdersSubject,
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

	mux := http.NewServeMux()
	runtime.mountDispatch(mux)

	req := httptest.NewRequestWithContext(
		ctx,
		http.MethodPost,
		crawldispatch.PathCrawlDispatch,
		strings.NewReader(`{"name":"docs","seeds":["https://example.org"],"maxPagesPerHost":-1}`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("dispatch status = %d, want 202; body=%s", rec.Code, rec.Body.String())
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
	if err := yacycrawlcontract.EnsureOrdersStream(
		ctx, js, yacycrawlcontract.OrdersStreamSpec{Subject: cfg.OrdersSubject},
	); err != nil {
		t.Fatalf("create orders stream: %v", err)
	}
}
