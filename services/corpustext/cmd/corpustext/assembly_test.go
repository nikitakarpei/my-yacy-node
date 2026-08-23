package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	corpustext "github.com/nikitakarpei/yacy-rwi-node/corpustext/cmd/corpustext"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

const (
	indexedDocumentPathPrefix = "/yacy_text_v1_en/_doc/"
	originURL                 = "http://origin.example/"
	originHTML                = `<html lang="en"><title>Hi</title><body>words here</body></html>`

	indexedDeadline  = 5 * time.Second
	indexedPollPause = 50 * time.Millisecond
)

func origin(t *testing.T) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(originHTML))
	}))
	t.Cleanup(server.Close)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse origin url: %v", err)
	}
	return parsed
}

type recordingElasticsearch struct {
	mu       sync.Mutex
	lastPath string
}

func (e *recordingElasticsearch) serve(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		e.lastPath = r.URL.Path
		e.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func (e *recordingElasticsearch) path() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastPath
}

func serviceConfig(
	scrapeRequestNATSURL, elasticsearchURL string,
	proxy *url.URL,
) corpustext.ServiceConfig {
	return corpustext.ServiceConfig{
		ScrapeRequestNATSURL:           scrapeRequestNATSURL,
		ScrapeRequestSubject:           corpustext.DefaultScrapeRequestSubject,
		ScrapeRequestDurable:           corpustext.DefaultScrapeRequestDurable,
		ProxyURL:                       proxy,
		UserAgent:                      corpustext.DefaultUserAgent,
		MaxBodyBytes:                   corpustext.DefaultScrapeMaxBodyBytes,
		FetchDeadline:                  time.Second,
		ScrapeRequestIntakeConcurrency: corpustext.DefaultScrapeRequestIntakeConcurrency,
		SearchIndexEngine:              corpustext.SearchIndexEngineElasticsearch,
		ElasticsearchURL:               elasticsearchURL,
		ElasticsearchIndex:             corpustext.DefaultIndexBaseName,
		Languages:                      []string{"en"},
		OpsAddr:                        "127.0.0.1:0",
	}
}

func TestRunServiceIndexesTheTextItScrapesFromAScrapeRequest(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	scrapeRequestNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(scrapeRequestNATSURL, elasticsearch.serve(t), origin(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scrapeRequestJetStream := natstestserver.ConnectJetStream(t, scrapeRequestNATSURL)
	createScrapeRequestsStream(t, scrapeRequestJetStream, cfg.ScrapeRequestSubject)

	runDone := make(chan error, 1)
	go func() { runDone <- corpustext.RunService(ctx, cfg) }()

	publishScrapeRequest(t, ctx, scrapeRequestJetStream, originURL)
	waitForIndexed(t, elasticsearch)

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

func waitForIndexed(t *testing.T, elasticsearch *recordingElasticsearch) {
	t.Helper()
	deadline := time.Now().Add(indexedDeadline)
	for time.Now().Before(deadline) {
		if strings.HasPrefix(elasticsearch.path(), indexedDocumentPathPrefix) {
			return
		}
		time.Sleep(indexedPollPause)
	}
	t.Fatalf(
		"elasticsearch never received the indexed document, last path = %q",
		elasticsearch.path(),
	)
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
	if _, err := js.Publish(ctx, corpustext.DefaultScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
	}
}

func TestRunServiceReturnsWhenOpsAddrCannotBind(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	scrapeRequestNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(scrapeRequestNATSURL, elasticsearch.serve(t), origin(t))
	cfg.OpsAddr = "127.0.0.1:99999"
	createScrapeRequestsStream(
		t, natstestserver.ConnectJetStream(t, scrapeRequestNATSURL), cfg.ScrapeRequestSubject,
	)

	if err := corpustext.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when ops address cannot bind")
	}
}

func TestRunServiceFailsWhenStreamMissing(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	cfg := serviceConfig(
		natstestserver.Start(t), elasticsearch.serve(t), origin(t),
	)

	if err := corpustext.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the scrape requests stream is not provisioned")
	}
}

func TestRunServiceFailsWhenCrawlNATSUnreachable(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	cfg := serviceConfig(
		"nats://127.0.0.1:1", elasticsearch.serve(t), origin(t),
	)

	if err := corpustext.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the crawl nats is unreachable")
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
