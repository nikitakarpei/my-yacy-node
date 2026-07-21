package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
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

func provisionBuckets(t *testing.T, js jetstream.JetStream, subject string) jetstream.ObjectStore {
	t.Helper()
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureOrdersStream(ctx, js, yacycrawlcontract.OrdersStreamSpec{
		Subject: subject,
	}); err != nil {
		t.Fatalf("ensure orders stream: %v", err)
	}
	if err := yacycrawlcontract.EnsureRedirectResolutionBucket(
		ctx, js, yacycrawlcontract.RedirectResolutionBucketSpec{},
	); err != nil {
		t.Fatalf("ensure redirect bucket: %v", err)
	}
	store, err := pagemarkdownstore.EnsureBucket(ctx, js)
	if err != nil {
		t.Fatalf("ensure markdown bucket: %v", err)
	}
	return store
}

func recallClient(t *testing.T, addr string) corpusrecallv1.RecallClient {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial recall: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return corpusrecallv1.NewRecallClient(conn)
}

func testConfig(natsURL, listenAddr string) ServiceConfig {
	return ServiceConfig{
		NATSURL:          natsURL,
		OrdersSubject:    DefaultOrdersSubject,
		ListenAddr:       listenAddr,
		OpsAddr:          "127.0.0.1:0",
		Deadline:         2 * time.Second,
		PollInterval:     20 * time.Millisecond,
		MaxInFlight:      DefaultMaxInFlight,
		MaxResponseBytes: DefaultMaxResponseBytes,
	}
}

func TestRunServiceRecallsStoredMarkdown(t *testing.T) {
	url := natstestserver.Start(t)
	js := natstestserver.ConnectJetStream(t, url)
	store := provisionBuckets(t, js, DefaultOrdersSubject)

	const canonicalURL = "https://example.com/"
	if _, err := store.PutBytes(
		context.Background(), pagemarkdownstore.ObjectName(canonicalURL), []byte("# Hi"),
	); err != nil {
		t.Fatalf("seed markdown: %v", err)
	}

	listenAddr := freeAddr(t)
	cfg := testConfig(url, listenAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- RunService(ctx, cfg) }()

	client := recallClient(t, listenAddr)
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

func TestRunServiceFailsWhenNATSUnreachable(t *testing.T) {
	cfg := testConfig("nats://127.0.0.1:1", freeAddr(t))
	if err := RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when nats is unreachable")
	}
}

func TestRunServiceFailsWhenOrdersStreamMissing(t *testing.T) {
	cfg := testConfig(natstestserver.Start(t), freeAddr(t))
	if err := RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when redirect bucket is not provisioned")
	}
}

func TestRunServiceFailsWhenListenAddrInvalid(t *testing.T) {
	url := natstestserver.Start(t)
	provisionBuckets(t, natstestserver.ConnectJetStream(t, url), DefaultOrdersSubject)
	cfg := testConfig(url, "127.0.0.1:99999")
	if err := RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected error when listen address cannot bind")
	}
}
