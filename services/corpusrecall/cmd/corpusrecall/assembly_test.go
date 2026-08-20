package main_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	corpusrecall "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/cmd/corpusrecall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/recallclienttest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release address: %v", err)
	}
	return addr
}

func provisionCrawlerState(t *testing.T, js jetstream.JetStream) {
	t.Helper()
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{corpusrecall.DefaultOrdersSubject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatalf("create orders stream: %v", err)
	}
	if _, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.RedirectResolutionBucketName,
	}); err != nil {
		t.Fatalf("create redirect resolution bucket: %v", err)
	}
	if _, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.DisposedPagesBucketName,
	}); err != nil {
		t.Fatalf("create disposed pages bucket: %v", err)
	}
}

func provisionPageMarkdownBucket(t *testing.T, js jetstream.JetStream) jetstream.ObjectStore {
	t.Helper()
	store, err := js.CreateOrUpdateObjectStore(context.Background(), jetstream.ObjectStoreConfig{
		Bucket: pagemarkdownstore.BucketName,
	})
	if err != nil {
		t.Fatalf("create page markdown bucket: %v", err)
	}
	return store
}

func testConfig(crawlNATSURL, pageMarkdownNATSURL, listenAddr string) corpusrecall.ServiceConfig {
	return corpusrecall.ServiceConfig{
		CrawlNATSURL:        crawlNATSURL,
		PageMarkdownNATSURL: pageMarkdownNATSURL,
		OrdersSubject:       corpusrecall.DefaultOrdersSubject,
		ListenAddr:          listenAddr,
		OpsAddr:             "127.0.0.1:0",
		RecallLimit:         2 * time.Second,
		PollInterval:        20 * time.Millisecond,
		MaxInFlight:         corpusrecall.DefaultMaxInFlight,
		MaxResponseBytes:    corpusrecall.DefaultMaxResponseBytes,
	}
}

func TestRunServiceRecallsStoredMarkdownFromItsOwnNATS(t *testing.T) {
	crawlURL := natstestserver.Start(t)
	pageMarkdownURL := natstestserver.Start(t)
	provisionCrawlerState(t, natstestserver.ConnectJetStream(t, crawlURL))
	store := provisionPageMarkdownBucket(t, natstestserver.ConnectJetStream(t, pageMarkdownURL))

	const canonicalURL = "https://example.com/"
	if _, err := store.PutBytes(
		context.Background(), pagemarkdownstore.ObjectName(canonicalURL), []byte("# Hi"),
	); err != nil {
		t.Fatalf("seed markdown: %v", err)
	}

	listenAddr := freeAddr(t)
	cfg := testConfig(crawlURL, pageMarkdownURL, listenAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- corpusrecall.RunService(ctx, cfg) }()

	client := recallclienttest.New(t, listenAddr)
	resp := recallUntilMarkdown(t, client, canonicalURL)
	if resp.GetRepresentations()[0].GetMarkdown().GetMarkdown() != "# Hi" {
		t.Errorf("markdown = %v", resp.GetRepresentations())
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

func recallUntilMarkdown(
	t *testing.T,
	client corpusrecallv1.RecallClient,
	url string,
) *corpusrecallv1.RecallResponse {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, err := client.Recall(ctx, &corpusrecallv1.RecallRequest{
			Url: url,
			Kinds: []corpusrecallv1.RepresentationKind{
				corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
			},
		})
		cancel()
		if err == nil && len(resp.GetRepresentations()) == 1 {
			return resp
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("recall never returned markdown")
	return nil
}

func TestRunServiceFailsWhenCrawlNATSUnreachable(t *testing.T) {
	cfg := testConfig("nats://127.0.0.1:1", natstestserver.Start(t), freeAddr(t))
	if err := corpusrecall.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the crawl nats is unreachable")
	}
}

func TestRunServiceFailsWhenPageMarkdownNATSUnreachable(t *testing.T) {
	url := natstestserver.Start(t)
	provisionCrawlerState(t, natstestserver.ConnectJetStream(t, url))
	cfg := testConfig(url, "nats://127.0.0.1:1", freeAddr(t))
	if err := corpusrecall.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when the page markdown nats is unreachable")
	}
}

func TestRunServiceFailsWhenPageMarkdownBucketMissing(t *testing.T) {
	url := natstestserver.Start(t)
	provisionCrawlerState(t, natstestserver.ConnectJetStream(t, url))
	cfg := testConfig(url, url, freeAddr(t))
	if err := corpusrecall.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when page markdown bucket is not provisioned")
	}
}

func TestRunServiceFailsWhenListenAddrInvalid(t *testing.T) {
	url := natstestserver.Start(t)
	js := natstestserver.ConnectJetStream(t, url)
	provisionCrawlerState(t, js)
	provisionPageMarkdownBucket(t, js)
	cfg := testConfig(url, url, "127.0.0.1:99999")
	if err := corpusrecall.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when listen address cannot bind")
	}
}
