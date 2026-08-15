package main_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/cmd/corpusmarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	storedDeadline  = 5 * time.Second
	storedPollPause = 50 * time.Millisecond
	storedReadLimit = 500 * time.Millisecond
)

func TestRunServiceStoresCrawledPageMarkdown(t *testing.T) {
	url := natstestserver.Start(t)
	cfg := corpusmarkdown.ServiceConfig{
		NATSURL:            url,
		CrawledPageSubject: corpusmarkdown.DefaultCrawledPageSubject,
		CrawledPageDurable: corpusmarkdown.DefaultCrawledPageDurable,
		Concurrency:        corpusmarkdown.DefaultConcurrency,
		OpsAddr:            "127.0.0.1:0",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	js := natstestserver.ConnectJetStream(t, url)
	createCrawledPageStream(t, js, cfg.CrawledPageSubject)

	runDone := make(chan error, 1)
	go func() { runDone <- corpusmarkdown.RunService(ctx, cfg) }()

	const canonicalURL = "https://example.com/"
	store, err := js.CreateOrUpdateObjectStore(ctx, jetstream.ObjectStoreConfig{
		Bucket: pagemarkdownstore.BucketName,
	})
	if err != nil {
		t.Fatalf("open object store: %v", err)
	}
	objectName := pagemarkdownstore.ObjectName(canonicalURL)

	publishMarkdown(t, ctx, js, canonicalURL, []byte("# Hi\n\nwords here"))
	waitForStored(t, ctx, store, objectName, []byte("# Hi\n\nwords here"))

	publishMarkdown(t, ctx, js, canonicalURL, []byte("# Hi again"))
	waitForStored(t, ctx, store, objectName, []byte("# Hi again"))

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

func publishMarkdown(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	canonicalURL string,
	markdown []byte,
) {
	t.Helper()
	data, err := yacycrawlcontract.MarshalPageMarkdownRepresentation(
		yacycrawlcontract.PageMarkdownRepresentation{
			PageReference: yacycrawlcontract.PageReference{CanonicalURL: canonicalURL},
			Markdown:      markdown,
		},
	)
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	if _, err := js.Publish(ctx, corpusmarkdown.DefaultCrawledPageSubject, data); err != nil {
		t.Fatalf("publish crawled page: %v", err)
	}
}

func waitForStored(
	t *testing.T,
	ctx context.Context,
	store jetstream.ObjectStore,
	name string,
	want []byte,
) {
	t.Helper()
	deadline := time.Now().Add(storedDeadline)
	for time.Now().Before(deadline) {
		if storedMarkdownMatches(ctx, store, name, want) {
			return
		}
		time.Sleep(storedPollPause)
	}
	t.Fatalf("markdown object %q never reached %q", name, want)
}

func storedMarkdownMatches(
	ctx context.Context,
	store jetstream.ObjectStore,
	name string,
	want []byte,
) bool {
	readCtx, cancel := context.WithTimeout(ctx, storedReadLimit)
	defer cancel()
	stored, err := store.GetBytes(readCtx, name)

	return err == nil && string(stored) == string(want)
}

func TestRunServiceReturnsWhenOpsAddrCannotBind(t *testing.T) {
	url := natstestserver.Start(t)
	cfg := corpusmarkdown.ServiceConfig{
		NATSURL:            url,
		CrawledPageSubject: corpusmarkdown.DefaultCrawledPageSubject,
		CrawledPageDurable: corpusmarkdown.DefaultCrawledPageDurable,
		Concurrency:        corpusmarkdown.DefaultConcurrency,
		OpsAddr:            "127.0.0.1:99999",
	}
	createCrawledPageStream(t, natstestserver.ConnectJetStream(t, url), cfg.CrawledPageSubject)

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when ops address cannot bind")
	}
}

func TestRunServiceFailsWhenStreamMissing(t *testing.T) {
	cfg := corpusmarkdown.ServiceConfig{
		NATSURL:            natstestserver.Start(t),
		CrawledPageSubject: corpusmarkdown.DefaultCrawledPageSubject,
		CrawledPageDurable: corpusmarkdown.DefaultCrawledPageDurable,
		Concurrency:        corpusmarkdown.DefaultConcurrency,
		OpsAddr:            "127.0.0.1:0",
	}

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when crawled page stream is not provisioned")
	}
}

func TestRunServiceFailsWhenNATSUnreachable(t *testing.T) {
	cfg := corpusmarkdown.ServiceConfig{
		NATSURL:            "nats://127.0.0.1:1",
		CrawledPageSubject: corpusmarkdown.DefaultCrawledPageSubject,
		CrawledPageDurable: corpusmarkdown.DefaultCrawledPageDurable,
		Concurrency:        corpusmarkdown.DefaultConcurrency,
		OpsAddr:            "127.0.0.1:0",
	}

	if err := corpusmarkdown.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when nats is unreachable")
	}
}

func createCrawledPageStream(t *testing.T, js jetstream.JetStream, subject string) {
	t.Helper()
	if _, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name: yacycrawlcontract.CrawledPageStreamName(
			yacycrawlcontract.PageRepresentationKindMarkdown,
		),
		Subjects:  []string{subject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   64,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		t.Fatalf("create crawled page stream: %v", err)
	}
}
