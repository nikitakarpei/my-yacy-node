package main_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	yacycrawler "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/cmd/yacycrawler"
)

func TestRunServiceProcessesOrderThenStops(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	crawlNATSURL := natstestserver.Start(t)
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:       crawlNATSURL,
		CrawlStateDir:      filepath.Join(t.TempDir(), "crawl-state"),
		CrawlOrdersSubject: yacycrawler.DefaultCrawlOrdersSubject,
		CrawlOrdersDurable: yacycrawler.DefaultCrawlOrdersDurable,
		ProxyURL:           proxy,
		FetchConcurrency:   2,
		MaxBodyBytes:       yacycrawler.DefaultMaxBodyBytes,
		FetchDeadline:      time.Second,
	}

	publishOrder(t, cfg.CrawlNATSURL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	listeners := yacycrawler.Listeners{Ops: listenerFor(t)}
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(ctx, cfg, listeners, registry); err != nil &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("yacycrawler.RunService: %v", err)
	}
}

func TestRunServiceRejectsBadCrawlNATSURL(t *testing.T) {
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:     "nats://127.0.0.1:1",
		FetchConcurrency: 2,
		FetchDeadline:    time.Second,
	}
	listeners := yacycrawler.Listeners{Ops: listenerFor(t)}
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(context.Background(), cfg, listeners, registry); err == nil {
		t.Fatal("unreachable nats should fail")
	}
}

func TestRunServiceRejectsACrawlStateDirectoryItCannotOpen(t *testing.T) {
	notADirectory := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notADirectory, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:       natstestserver.Start(t),
		CrawlStateDir:      filepath.Join(notADirectory, "crawl-state"),
		CrawlOrdersSubject: yacycrawler.DefaultCrawlOrdersSubject,
		FetchConcurrency:   2,
		FetchDeadline:      time.Second,
	}
	listeners := yacycrawler.Listeners{Ops: listenerFor(t)}
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(context.Background(), cfg, listeners, registry); err == nil {
		t.Fatal("a crawl state directory the crawler cannot open should fail")
	}
}

func publishOrder(t *testing.T, crawlNATSURL string) {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, crawlNATSURL)
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{yacycrawler.DefaultCrawlOrdersSubject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	payload, err := yacycrawlcontract.MarshalCrawlOrder(yacycrawlcontract.CrawlOrder{
		OrderID: "o1",
		Profile: yacycrawlcontract.CrawlProfile{
			Scope: yacycrawlcontract.ScopeWide, URLMustMatch: yacycrawlcontract.MatchAll,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://origin.example/"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(ctx, yacycrawler.DefaultCrawlOrdersSubject, payload); err != nil {
		t.Fatal(err)
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
