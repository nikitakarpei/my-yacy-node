package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	corpustext "github.com/nikitakarpei/yacy-rwi-node/corpustext/cmd/corpustext"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	indexedDocumentPathPrefix = "/yacy_text_v1_en/_doc/"
	offeredPageURL            = "http://origin.example/"
	offeredPageHTML           = `<html lang="en"><title>Hi</title><body>words here</body></html>`

	indexedDeadline  = 5 * time.Second
	indexedPollPause = 50 * time.Millisecond
)

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

func serviceConfig(pageOfferNATSURL, elasticsearchURL string) corpustext.ServiceConfig {
	return corpustext.ServiceConfig{
		PageOfferNATSURL:           pageOfferNATSURL,
		PageOfferDurable:           corpustext.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: corpustext.DefaultPageOfferIntakeConcurrency,
		SearchIndexEngine:          corpustext.SearchIndexEngineElasticsearch,
		ElasticsearchURL:           elasticsearchURL,
		ElasticsearchIndex:         corpustext.DefaultIndexBaseName,
		Languages:                  []string{"en"},
		OpsAddr:                    "127.0.0.1:0",
	}
}

func TestRunServiceIndexesTheTextOfAnOfferedPage(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(pageOfferNATSURL, elasticsearch.serve(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pageOfferJetStream := natstestserver.ConnectJetStream(t, pageOfferNATSURL)
	createScrapePageOffersStream(t, pageOfferJetStream)
	keptPages := subscribeToKeptPages(t, pageOfferNATSURL)

	runDone := make(chan error, 1)
	go func() { runDone <- corpustext.RunService(ctx, cfg) }()

	waitForPageOfferDurable(ctx, t, pageOfferJetStream, cfg.PageOfferDurable)
	publishOfferedPage(ctx, t, pageOfferJetStream)
	waitForIndexed(t, elasticsearch)
	waitForKeptPageReceipt(t, keptPages)

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

func waitForPageOfferDurable(
	ctx context.Context,
	t *testing.T,
	js jetstream.JetStream,
	durable string,
) {
	t.Helper()
	deadline := time.Now().Add(indexedDeadline)
	for time.Now().Before(deadline) {
		stream, err := js.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
		if err == nil {
			if _, err := stream.Consumer(ctx, durable); err == nil {
				return
			}
		}
		time.Sleep(indexedPollPause)
	}
	t.Fatalf("the service never created the %q durable", durable)
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

func subscribeToKeptPages(t *testing.T, natsURL string) chan *nats.Msg {
	t.Helper()
	conn := natstestserver.Connect(t, natsURL)
	received := make(chan *nats.Msg, 1)
	subject := pagescrapecontract.KeptPageSubjectOf(
		canonicalurltest.CanonicalURLOf(t, offeredPageURL),
	)
	subscription, err := conn.ChanSubscribe(subject, received)
	if err != nil {
		t.Fatalf("subscribe to kept pages: %v", err)
	}
	t.Cleanup(func() { _ = subscription.Unsubscribe() })
	return received
}

func waitForKeptPageReceipt(t *testing.T, keptPages chan *nats.Msg) {
	t.Helper()
	select {
	case message := <-keptPages:
		kept, err := pagescrapecontract.UnmarshalKeptPage(message.Data)
		if err != nil {
			t.Fatalf("unmarshal kept page: %v", err)
		}
		if kept.PageURL != canonicalurltest.CanonicalURLOf(t, offeredPageURL) {
			t.Errorf("kept page url = %q", kept.PageURL)
		}
	case <-time.After(indexedDeadline):
		t.Fatal("no kept page receipt arrived")
	}
}

func publishOfferedPage(ctx context.Context, t *testing.T, js jetstream.JetStream) {
	t.Helper()
	pageURL := canonicalurltest.CanonicalURLOf(t, offeredPageURL)
	data, err := pagescrapecontract.MarshalOfferedPage(pagescrapecontract.OfferedPage{
		PageURL:     pageURL,
		LandedURL:   pageURL,
		ContentType: "text/html",
		Body:        []byte(offeredPageHTML),
	})
	if err != nil {
		t.Fatalf("marshal offered page: %v", err)
	}
	if _, err := js.Publish(ctx, pagescrapecontract.OfferedPageSubject, data); err != nil {
		t.Fatalf("publish offered page: %v", err)
	}
}

func TestRunServiceReturnsWhenOpsAddrCannotBind(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(pageOfferNATSURL, elasticsearch.serve(t))
	cfg.OpsAddr = "127.0.0.1:99999"
	createScrapePageOffersStream(t, natstestserver.ConnectJetStream(t, pageOfferNATSURL))

	if err := corpustext.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when ops address cannot bind")
	}
}

func TestRunServiceFailsWhenPageOffersStreamMissing(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	cfg := serviceConfig(natstestserver.Start(t), elasticsearch.serve(t))

	if err := corpustext.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page offers stream is not provisioned")
	}
}

func TestRunServiceFailsWhenPageOfferNATSUnreachable(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	cfg := serviceConfig("nats://127.0.0.1:1", elasticsearch.serve(t))

	if err := corpustext.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page offer nats is unreachable")
	}
}

func createScrapePageOffersStream(t *testing.T, js jetstream.JetStream) {
	t.Helper()
	if _, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name: pagescrapecontract.ScrapePageOffersStreamName,
		Subjects: []string{
			pagescrapecontract.OfferedPageSubject,
			pagescrapecontract.ScrapeFailureSubject,
		},
		Retention: jetstream.InterestPolicy,
		MaxMsgs:   64,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		t.Fatalf("create scrape page offers stream: %v", err)
	}
}
