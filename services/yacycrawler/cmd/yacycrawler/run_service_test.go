package main_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	yacycrawler "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/cmd/yacycrawler"
)

const crawlWait = 10 * time.Second

func TestRunServiceProcessesOrderThenStops(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	crawlNATSURL := natstestserver.Start(t)
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:       crawlNATSURL,
		CrawlStateRedis:    &redis.Options{Addr: miniredis.RunT(t).Addr()},
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

func TestRunServiceCrawlsOnItsOwnBrokerWhenNoCrawlNATSURLIsSet(t *testing.T) {
	cfg := yacycrawler.ServiceConfig{
		CrawlStateRedis:    &redis.Options{Addr: miniredis.RunT(t).Addr()},
		CrawlOrdersSubject: yacycrawler.DefaultCrawlOrdersSubject,
		CrawlOrdersDurable: yacycrawler.DefaultCrawlOrdersDurable,
		ProxyURL:           originServing(t, "<html lang=\"en\"><body>words here</body></html>"),
		FetchConcurrency:   2,
		MaxBodyBytes:       yacycrawler.DefaultMaxBodyBytes,
		FetchDeadline:      time.Second,
		NATSServer:         jetstreamserver.Config{Addr: freeAddrFor(t), StoreDir: t.TempDir()},
	}
	crawlNATSURL := "nats://" + cfg.NATSServer.Addr
	broker := brokerAwaiting(t, crawlNATSURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listeners := yacycrawler.Listeners{Ops: listenerFor(t)}
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacycrawler.RunService(ctx, cfg, listeners, prometheus.NewRegistry())
	}()

	pages := crawledPageConsumer(t, broker)
	publishOrder(t, crawlNATSURL)

	if _, err := pages.Next(jetstream.FetchMaxWait(crawlWait)); err != nil {
		t.Fatalf("await crawled page: %v", err)
	}

	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("yacycrawler.RunService: %v", err)
		}
	case <-time.After(crawlWait):
		t.Fatal("service did not shut down after cancel")
	}
}

func originServing(t *testing.T, body string) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	proxy, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse origin url: %v", err)
	}
	return proxy
}

func freeAddrFor(t *testing.T) string {
	t.Helper()
	listener := listenerFor(t)
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("free %s: %v", addr, err)
	}
	return addr
}

func brokerAwaiting(t *testing.T, natsURL string) jetstream.JetStream {
	t.Helper()
	connection, err := nats.Connect(
		natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(connection.Close)
	broker, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return broker
}

func crawledPageConsumer(t *testing.T, broker jetstream.JetStream) jetstream.Consumer {
	t.Helper()
	deadline := time.Now().Add(crawlWait)
	for {
		consumer, err := broker.CreateOrUpdateConsumer(
			context.Background(),
			yacycrawlcontract.CrawledPagesStreamName,
			jetstream.ConsumerConfig{Durable: "runservicetest", AckPolicy: jetstream.AckNonePolicy},
		)
		if err == nil {
			return consumer
		}
		if time.Now().After(deadline) {
			t.Fatalf("open crawled page consumer: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestRunServiceRejectsAnOwnBrokerThatCannotStart(t *testing.T) {
	cfg := yacycrawler.ServiceConfig{
		FetchConcurrency: 2,
		FetchDeadline:    time.Second,
		NATSServer:       jetstreamserver.Config{Addr: "no-port", StoreDir: t.TempDir()},
	}
	listeners := yacycrawler.Listeners{Ops: listenerFor(t)}
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(context.Background(), cfg, listeners, registry); err == nil {
		t.Fatal("a broker that cannot start should fail")
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

func TestRunServiceRejectsAnUnreachableCrawlStateServer(t *testing.T) {
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:       natstestserver.Start(t),
		CrawlStateRedis:    &redis.Options{Addr: "127.0.0.1:1"},
		CrawlOrdersSubject: yacycrawler.DefaultCrawlOrdersSubject,
		FetchConcurrency:   2,
		FetchDeadline:      time.Second,
	}
	listeners := yacycrawler.Listeners{Ops: listenerFor(t)}
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(context.Background(), cfg, listeners, registry); err == nil {
		t.Fatal("a Redis address that does not answer should fail")
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
