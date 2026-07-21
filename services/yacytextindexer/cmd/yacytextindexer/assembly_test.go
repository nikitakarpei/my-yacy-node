package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestRunServiceIndexesCrawledPageIntoElasticsearch(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	elasticsearch := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			gotPath = r.URL.Path
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		}),
	)
	defer elasticsearch.Close()

	url := natstestserver.Start(t)
	cfg := ServiceConfig{
		NATSURL:            url,
		CrawledPageSubject: "yacy.crawl.page.text",
		CrawledPageDurable: DefaultCrawledPageDurable,
		Concurrency:        DefaultConcurrency,
		SearchIndexEngine:  SearchIndexEngineElasticsearch,
		ElasticsearchURL:   elasticsearch.URL,
		ElasticsearchIndex: "yacy-text",
		OpsAddr:            "127.0.0.1:0",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan error, 1)
	js := natstestserver.ConnectJetStream(t, url)
	createCrawledPageStream(t, js, cfg.CrawledPageSubject)

	go func() { runDone <- RunService(ctx, cfg) }()

	data, err := yacycrawlcontract.MarshalPageTextRepresentation(
		yacycrawlcontract.PageTextRepresentation{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: "https://example.com/",
				Title:        "Hi",
			},
			Text: []byte("words here"),
		},
	)
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	if _, err := js.Publish(ctx, cfg.CrawledPageSubject, data); err != nil {
		t.Fatalf("publish crawled page: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		path := gotPath
		mu.Unlock()
		if path == "/yacy-text/_doc/0f115db062b7c0dd030b16878c99dea5c354b49dc37b38eb8846179c7783e9d7" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	mu.Lock()
	path := gotPath
	mu.Unlock()
	if path != "/yacy-text/_doc/0f115db062b7c0dd030b16878c99dea5c354b49dc37b38eb8846179c7783e9d7" {
		t.Fatalf("elasticsearch never received the indexed document, last path = %q", path)
	}

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

func TestRunServiceReturnsWhenOpsAddrCannotBind(t *testing.T) {
	elasticsearch := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	)
	defer elasticsearch.Close()

	url := natstestserver.Start(t)
	cfg := ServiceConfig{
		NATSURL:            url,
		CrawledPageSubject: "yacy.crawl.page.text",
		CrawledPageDurable: DefaultCrawledPageDurable,
		Concurrency:        DefaultConcurrency,
		SearchIndexEngine:  SearchIndexEngineElasticsearch,
		ElasticsearchURL:   elasticsearch.URL,
		ElasticsearchIndex: "yacy-text",
		OpsAddr:            "127.0.0.1:99999",
	}
	createCrawledPageStream(t, natstestserver.ConnectJetStream(t, url), cfg.CrawledPageSubject)

	err := RunService(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when ops address cannot bind")
	}
}

func createCrawledPageStream(t *testing.T, js jetstream.JetStream, subject string) {
	t.Helper()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		context.Background(),
		js,
		yacycrawlcontract.PageRepresentationKindText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: subject, MaxMsgs: 64},
	); err != nil {
		t.Fatalf("create crawled page stream: %v", err)
	}
}
