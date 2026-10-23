package corpusmarkdownassembly_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/assembly"
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	offeredPageURL  = "http://origin.example/"
	offeredPageHTML = `<html lang="en"><title>Hi</title><body>words here</body></html>`

	storedDeadline  = 5 * time.Second
	storedPollPause = 50 * time.Millisecond
)

func objectStoreServer(t *testing.T, objects *s3mem.Backend) string {
	t.Helper()
	server := httptest.NewServer(gofakes3.New(objects).Server())
	t.Cleanup(server.Close)
	return server.URL
}

func objectStoreRefusingEverything(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>refused</Message></Error>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func serviceConfig(pageOfferNATSURL, objectStoreURL string) corpusmarkdown.ServiceConfig {
	return corpusmarkdown.ServiceConfig{
		PageOfferNATSURL: pageOfferNATSURL,
		PageMarkdownObjectStore: &pagemarkdowncorporas3.ObjectStore{
			Endpoint:        objectStoreURL,
			Region:          corpusmarkdown.DefaultPageMarkdownS3Region,
			AccessKeyID:     "corpus",
			SecretAccessKey: "corpus-secret",
		},
		PageMarkdownBucket:         corpusmarkdown.DefaultPageMarkdownBucket,
		PageOfferDurable:           corpusmarkdown.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: corpusmarkdown.DefaultPageOfferIntakeConcurrency,
	}
}

func TestRunServiceStoresTheMarkdownOfAnOfferedPage(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	objects := s3mem.New()
	cfg := serviceConfig(pageOfferNATSURL, objectStoreServer(t, objects))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pageOfferJetStream := natstestserver.ConnectJetStream(t, pageOfferNATSURL)
	createScrapePageOffersStream(t, pageOfferJetStream)
	keptPages := subscribeToKeptPages(t, pageOfferNATSURL)

	listeners := corpusmarkdown.Listeners{Recall: listenerFor(t)}
	runDone := make(chan error, 1)
	go func() { runDone <- corpusmarkdown.RunService(ctx, cfg, listeners, prometheus.NewRegistry()) }()

	waitForPageOfferDurable(ctx, t, pageOfferJetStream, cfg.PageOfferDurable)
	publishOfferedPage(ctx, t, pageOfferJetStream)
	waitForStored(t, objects, cfg.PageMarkdownBucket,
		pagemarkdownstore.ObjectNameOf(canonicalurltest.CanonicalURLOf(t, offeredPageURL)),
		"words here",
	)
	waitForKeptPageReceipt(t, keptPages)

	cancel()
	waitForShutdown(t, runDone)
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

func TestRunServiceRecallsTheMarkdownItStoredInADirectory(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := corpusmarkdown.ServiceConfig{
		PageOfferNATSURL:           pageOfferNATSURL,
		PageMarkdownDirectory:      t.TempDir(),
		PageOfferDurable:           corpusmarkdown.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: corpusmarkdown.DefaultPageOfferIntakeConcurrency,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pageOfferJetStream := natstestserver.ConnectJetStream(t, pageOfferNATSURL)
	createScrapePageOffersStream(t, pageOfferJetStream)

	listeners := corpusmarkdown.Listeners{Recall: listenerFor(t)}
	runDone := make(chan error, 1)
	go func() { runDone <- corpusmarkdown.RunService(ctx, cfg, listeners, prometheus.NewRegistry()) }()

	waitForPageOfferDurable(ctx, t, pageOfferJetStream, cfg.PageOfferDurable)
	publishOfferedPage(ctx, t, pageOfferJetStream)
	waitForRecalledMarkdown(ctx, t, listeners.Recall.Addr().String(), "words here")

	cancel()
	waitForShutdown(t, runDone)
}

func TestRunServiceFailsWhenThePageMarkdownDirectoryCannotBeCreated(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	createScrapePageOffersStream(t, natstestserver.ConnectJetStream(t, pageOfferNATSURL))
	occupied := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(occupied, nil, 0o600); err != nil {
		t.Fatalf("occupy the path: %v", err)
	}
	cfg := corpusmarkdown.ServiceConfig{
		PageOfferNATSURL:           pageOfferNATSURL,
		PageMarkdownDirectory:      filepath.Join(occupied, "markdown"),
		PageOfferDurable:           corpusmarkdown.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: corpusmarkdown.DefaultPageOfferIntakeConcurrency,
	}
	listeners := corpusmarkdown.Listeners{Recall: listenerFor(t)}

	if err := corpusmarkdown.RunService(
		context.Background(), cfg, listeners, prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("expected error when the page markdown directory cannot be created")
	}
}

func waitForRecalledMarkdown(ctx context.Context, t *testing.T, recallAddress, want string) {
	t.Helper()
	recallURL := "http://" + recallAddress + pagemarkdownstore.RecalledPagePath + "?" +
		url.Values{pagemarkdownstore.RequestedURLQueryParam: {offeredPageURL}}.Encode()
	deadline := time.Now().Add(storedDeadline)
	for time.Now().Before(deadline) {
		if recalledMarkdownCarries(ctx, recallURL, want) {
			return
		}
		time.Sleep(storedPollPause)
	}
	t.Fatalf("the recalled markdown of %q never carried %q", offeredPageURL, want)
}

func recalledMarkdownCarries(ctx context.Context, recallURL, want string) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, recallURL, http.NoBody)
	if err != nil {
		return false
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false
	}
	defer func() { _ = response.Body.Close() }()
	var recalled pagemarkdownstore.RecalledPage
	if response.StatusCode != http.StatusOK ||
		json.NewDecoder(response.Body).Decode(&recalled) != nil {
		return false
	}
	return strings.Contains(recalled.Markdown, want)
}

func waitForPageOfferDurable(
	ctx context.Context,
	t *testing.T,
	js jetstream.JetStream,
	durable string,
) {
	t.Helper()
	deadline := time.Now().Add(storedDeadline)
	for time.Now().Before(deadline) {
		stream, err := js.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
		if err == nil {
			if _, err := stream.Consumer(ctx, durable); err == nil {
				return
			}
		}
		time.Sleep(storedPollPause)
	}
	t.Fatalf("the service never created the %q durable", durable)
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
	if err := conn.Flush(); err != nil {
		t.Fatalf("flush the kept page subscription: %v", err)
	}
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
	case <-time.After(storedDeadline):
		t.Fatal("no kept page receipt arrived")
	}
}

func waitForStored(
	t *testing.T,
	objects *s3mem.Backend,
	bucket, name, want string,
) {
	t.Helper()
	deadline := time.Now().Add(storedDeadline)
	for time.Now().Before(deadline) {
		if storedMarkdownCarries(objects, bucket, name, want) {
			return
		}
		time.Sleep(storedPollPause)
	}
	t.Fatalf("markdown object %q never carried %q", name, want)
}

func storedMarkdownCarries(objects *s3mem.Backend, bucket, name, want string) bool {
	object, err := objects.GetObject(bucket, name, nil)
	if err != nil {
		return false
	}
	defer func() { _ = object.Contents.Close() }()
	stored, err := io.ReadAll(object.Contents)

	return err == nil && strings.Contains(string(stored), want)
}

func TestRunServiceFailsWhenPageOffersStreamMissing(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t), objectStoreServer(t, s3mem.New()))
	listeners := corpusmarkdown.Listeners{Recall: listenerFor(t)}

	if err := corpusmarkdown.RunService(
		context.Background(), cfg, listeners, prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("expected error when the page offers stream is not provisioned")
	}
}

func TestRunServiceFailsWhenPageOfferNATSUnreachable(t *testing.T) {
	cfg := serviceConfig("nats://127.0.0.1:1", objectStoreServer(t, s3mem.New()))
	listeners := corpusmarkdown.Listeners{Recall: listenerFor(t)}

	if err := corpusmarkdown.RunService(
		context.Background(), cfg, listeners, prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("expected error when the page offer nats is unreachable")
	}
}

func TestRunServiceFailsWhenPageMarkdownObjectStoreRefuses(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(pageOfferNATSURL, objectStoreRefusingEverything(t))
	createScrapePageOffersStream(t, natstestserver.ConnectJetStream(t, pageOfferNATSURL))
	listeners := corpusmarkdown.Listeners{Recall: listenerFor(t)}

	if err := corpusmarkdown.RunService(
		context.Background(), cfg, listeners, prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("expected error when the page markdown object store refuses")
	}
}

func TestListenersForListensOnTheRecallAddress(t *testing.T) {
	listeners, err := corpusmarkdown.ListenersFor(
		t.Context(), corpusmarkdown.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Recall.Close() })
	if listeners.Recall.Addr().String() == "127.0.0.1:0" {
		t.Errorf("recall listener = %s, want a bound port", listeners.Recall.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := corpusmarkdown.ListenersFor(
		t.Context(), corpusmarkdown.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable recall address")
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

func listenerFor(t *testing.T) net.Listener {
	t.Helper()

	listener, err := servergroup.Listen(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}
