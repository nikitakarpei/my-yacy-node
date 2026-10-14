package main_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/cmd/corpusmarkdown"
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
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
		PageMarkdownObjectStore: pagemarkdowncorporas3.ObjectStore{
			Endpoint:        objectStoreURL,
			Region:          corpusmarkdown.DefaultPageMarkdownS3Region,
			AccessKeyID:     "corpus",
			SecretAccessKey: "corpus-secret",
		},
		PageMarkdownBucket:         corpusmarkdown.DefaultPageMarkdownBucket,
		PageOfferDurable:           corpusmarkdown.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: corpusmarkdown.DefaultPageOfferIntakeConcurrency,
		ListenAddr:                 "127.0.0.1:0",
		OpsAddr:                    "127.0.0.1:0",
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

	runDone := make(chan error, 1)
	go func() { runDone <- corpusmarkdown.RunService(ctx, cfg) }()

	waitForPageOfferDurable(ctx, t, pageOfferJetStream, cfg.PageOfferDurable)
	publishOfferedPage(ctx, t, pageOfferJetStream)
	waitForStored(t, objects, cfg.PageMarkdownBucket,
		pagemarkdownstore.ObjectNameOf(canonicalurltest.CanonicalURLOf(t, offeredPageURL)),
		"words here",
	)
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

func TestRunServiceReturnsWhenOpsAddrCannotBind(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(pageOfferNATSURL, objectStoreServer(t, s3mem.New()))
	cfg.OpsAddr = "127.0.0.1:99999"
	createScrapePageOffersStream(t, natstestserver.ConnectJetStream(t, pageOfferNATSURL))

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when ops address cannot bind")
	}
}

func TestRunServiceFailsWhenPageOffersStreamMissing(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t), objectStoreServer(t, s3mem.New()))

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page offers stream is not provisioned")
	}
}

func TestRunServiceFailsWhenPageOfferNATSUnreachable(t *testing.T) {
	cfg := serviceConfig("nats://127.0.0.1:1", objectStoreServer(t, s3mem.New()))

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page offer nats is unreachable")
	}
}

func TestRunServiceFailsWhenPageMarkdownObjectStoreRefuses(t *testing.T) {
	pageOfferNATSURL := natstestserver.Start(t)
	cfg := serviceConfig(pageOfferNATSURL, objectStoreRefusingEverything(t))
	createScrapePageOffersStream(t, natstestserver.ConnectJetStream(t, pageOfferNATSURL))

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page markdown object store refuses")
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
