package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/cmd/corpusmarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

const (
	storedDeadline  = 5 * time.Second
	storedPollPause = 50 * time.Millisecond
	storedReadLimit = 500 * time.Millisecond

	originURL = "http://origin.example/"
)

func originServing(t *testing.T, body string) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse origin url: %v", err)
	}
	return parsed
}

func serviceConfig(
	scrapeRequestNATSURL, pageMarkdownURL string,
	proxy *url.URL,
) corpusmarkdown.ServiceConfig {
	return corpusmarkdown.ServiceConfig{
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		PageMarkdownNATSURL:  pageMarkdownURL,
		ScrapeRequestSubject: corpusmarkdown.DefaultScrapeRequestSubject,
		ScrapeRequestDurable: corpusmarkdown.DefaultScrapeRequestDurable,
		ProxyURL:             proxy,
		UserAgent:            corpusmarkdown.DefaultUserAgent,
		MaxBodyBytes:         corpusmarkdown.DefaultMaxBodyBytes,
		FetchDeadline:        time.Second,
		Concurrency:          corpusmarkdown.DefaultConcurrency,
		OpsAddr:              "127.0.0.1:0",
	}
}

func TestRunServiceStoresTheMarkdownItScrapesFromAScrapeRequest(t *testing.T) {
	scrapeRequestNATSURL := natstestserver.Start(t)
	pageMarkdownURL := natstestserver.Start(t)
	proxy := originServing(t, "<html lang=\"en\"><title>Hi</title><body>words here</body></html>")
	cfg := serviceConfig(scrapeRequestNATSURL, pageMarkdownURL, proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scrapeRequestJetStream := natstestserver.ConnectJetStream(t, scrapeRequestNATSURL)
	pageMarkdownJetStream := natstestserver.ConnectJetStream(t, pageMarkdownURL)
	createScrapeRequestsStream(t, scrapeRequestJetStream, cfg.ScrapeRequestSubject)

	runDone := make(chan error, 1)
	go func() { runDone <- corpusmarkdown.RunService(ctx, cfg) }()

	store, err := pageMarkdownJetStream.CreateOrUpdateObjectStore(
		ctx,
		jetstream.ObjectStoreConfig{Bucket: pagemarkdownstore.BucketName},
	)
	if err != nil {
		t.Fatalf("open object store: %v", err)
	}

	publishScrapeRequest(t, ctx, scrapeRequestJetStream, originURL)
	waitForStored(t, ctx, store,
		pagemarkdownstore.ObjectName(canonicalurltest.CanonicalURLOf(t, originURL)), "words here")

	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("run: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("service did not shut down after cancel")
	}
}

func publishScrapeRequest(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	canonicalURL string,
) {
	t.Helper()
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, canonicalURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	if _, err := js.Publish(ctx, corpusmarkdown.DefaultScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
	}
}

func waitForStored(
	t *testing.T,
	ctx context.Context,
	store jetstream.ObjectStore,
	name, want string,
) {
	t.Helper()
	deadline := time.Now().Add(storedDeadline)
	for time.Now().Before(deadline) {
		if storedMarkdownCarries(ctx, store, name, want) {
			return
		}
		time.Sleep(storedPollPause)
	}
	t.Fatalf("markdown object %q never carried %q", name, want)
}

func storedMarkdownCarries(
	ctx context.Context,
	store jetstream.ObjectStore,
	name, want string,
) bool {
	readCtx, cancel := context.WithTimeout(ctx, storedReadLimit)
	defer cancel()
	stored, err := store.GetBytes(readCtx, name)

	return err == nil && strings.Contains(string(stored), want)
}

func TestRunServiceReturnsWhenOpsAddrCannotBind(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(natsURL, natsURL, originServing(t, "<html></html>"))
	cfg.OpsAddr = "127.0.0.1:99999"
	createScrapeRequestsStream(
		t,
		natstestserver.ConnectJetStream(t, natsURL),
		cfg.ScrapeRequestSubject,
	)

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when ops address cannot bind")
	}
}

func TestRunServiceFailsWhenStreamMissing(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(natsURL, natsURL, originServing(t, "<html></html>"))

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the scrape requests stream is not provisioned")
	}
}

func TestRunServiceFailsWhenCrawlNATSUnreachable(t *testing.T) {
	cfg := serviceConfig(
		"nats://127.0.0.1:1",
		natstestserver.Start(t),
		originServing(t, "<html></html>"),
	)

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the crawl nats is unreachable")
	}
}

func TestRunServiceFailsWhenPageMarkdownNATSUnreachable(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(natsURL, "nats://127.0.0.1:1", originServing(t, "<html></html>"))
	createScrapeRequestsStream(
		t,
		natstestserver.ConnectJetStream(t, natsURL),
		cfg.ScrapeRequestSubject,
	)

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page markdown nats is unreachable")
	}
}

func createScrapeRequestsStream(t *testing.T, js jetstream.JetStream, subject string) {
	t.Helper()
	if _, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:      scraperequestcontract.ScrapeRequestsStreamName,
		Subjects:  []string{subject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   64,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		t.Fatalf("create scrape requests stream: %v", err)
	}
}
