package main_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
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
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	indexedDocumentPathPrefix = "/yacy_text_v1_en/_doc/"
	offeredPageURL            = "http://origin.example/"
	offeredPageHTML           = `<html lang="en"><title>Hi</title><body>words here</body></html>`

	indexedDeadline  = 5 * time.Second
	indexedPollPause = 50 * time.Millisecond

	answeredSearch = `{"hits":{"hits":[{
		"_source":{"title":"Hi","url":"http://origin.example/"},
		"highlight":{"content":["words here"]}
	}]}}`
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
		if strings.HasSuffix(r.URL.Path, "/_search") {
			_, _ = io.WriteString(w, answeredSearch)
			return
		}
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, "{}")
			return
		}
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
	go func() { runDone <- corpustext.RunService(ctx, cfg, listenersFor(t)) }()

	waitForPageOfferDurable(ctx, t, pageOfferJetStream, cfg.PageOfferDurable)
	publishOfferedPage(ctx, t, pageOfferJetStream)
	waitForIndexed(t, elasticsearch)
	waitForKeptPageReceipt(t, keptPages)

	cancel()
	waitForShutdown(t, runDone)
}

func TestRunServiceAnswersASearchForAnOfferedPageFromBleve(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(pageOfferNATSURL, "")
	cfg.SearchIndexEngine = corpustext.SearchIndexEngineBleve
	cfg.BleveDirectory = t.TempDir()
	listeners := listenersFor(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pageOfferJetStream := natstestserver.ConnectJetStream(t, pageOfferNATSURL)
	createScrapePageOffersStream(t, pageOfferJetStream)
	keptPages := subscribeToKeptPages(t, pageOfferNATSURL)

	runDone := make(chan error, 1)
	go func() { runDone <- corpustext.RunService(ctx, cfg, listeners) }()

	waitForPageOfferDurable(ctx, t, pageOfferJetStream, cfg.PageOfferDurable)
	publishOfferedPage(ctx, t, pageOfferJetStream)
	waitForKeptPageReceipt(t, keptPages)

	matches := matchesFrom(t, "http://"+listeners.Search.Addr().String())
	if len(matches.Matches) != 1 || matches.Matches[0].URL != offeredPageURL ||
		matches.Matches[0].Title != "Hi" {
		t.Errorf("matches = %+v", matches.Matches)
	}

	cancel()
	waitForShutdown(t, runDone)
}

func TestRunServiceAnswersSearchesWhileThePageOffersAreUnavailable(t *testing.T) {
	elasticsearch := &recordingElasticsearch{}
	cfg := serviceConfig("nats://127.0.0.1:1", elasticsearch.serve(t))
	listeners := listenersFor(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- corpustext.RunService(ctx, cfg, listeners) }()

	matches := matchesFrom(t, "http://"+listeners.Search.Addr().String())
	want := searchdocument.PageMatch{
		URL: "http://origin.example/", Title: "Hi", Snippet: "words here",
	}
	if len(matches.Matches) != 1 || matches.Matches[0] != want {
		t.Errorf("matches = %+v, want %+v", matches.Matches, want)
	}

	cancel()
	waitForShutdown(t, runDone)
}

func TestRunServiceFailsWhenTheSearchIndexCannotBeOpened(t *testing.T) {
	failingElasticsearch := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	))
	t.Cleanup(failingElasticsearch.Close)
	cfg := serviceConfig(natstestserver.Start(t), failingElasticsearch.URL)

	if err := corpustext.RunService(context.Background(), cfg, listenersFor(t)); err == nil {
		t.Fatal("expected error when the search index cannot be opened")
	}
}

func matchesFrom(t *testing.T, searchURL string) searchdocument.PageMatches {
	t.Helper()
	target := searchURL + searchdocument.SearchPath + "?" + searchdocument.SearchQueryParam + "=words"
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("build search request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("search status = %d", response.StatusCode)
	}
	var matches searchdocument.PageMatches
	if err := json.NewDecoder(response.Body).Decode(&matches); err != nil {
		t.Fatalf("decode matches: %v", err)
	}
	return matches
}

func waitForShutdown(t *testing.T, runDone chan error) {
	t.Helper()
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

func listenersFor(t *testing.T) corpustext.Listeners {
	t.Helper()
	return corpustext.Listeners{Search: listenerFor(t), Ops: listenerFor(t)}
}

func listenerFor(t *testing.T) net.Listener {
	t.Helper()

	listener, err := servergroup.Listen(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}
