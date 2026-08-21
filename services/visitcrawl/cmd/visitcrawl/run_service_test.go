package main_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	visitcrawl "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/cmd/visitcrawl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestRunServiceRedirectsAndPlacesOrder(t *testing.T) {
	crawlNATSURL := natstestserver.Start(t)
	cfg := visitcrawl.ServiceConfig{
		CrawlNATSURL:  crawlNATSURL,
		OrdersSubject: visitcrawl.DefaultOrdersSubject,
		ListenAddr:    freeAddr(t),
		OpsAddr:       freeAddr(t),
		OrderTimeout:  visitcrawl.DefaultOrderTimeout,
		MaxInFlight:   visitcrawl.DefaultMaxInFlight,
		MaxBodyBytes:  visitcrawl.DefaultMaxBodyBytes,
		LinkSecret:    "shared-secret",
		CrawlProfile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxPagesPerHost: visitcrawl.DefaultCrawlMaxPagesPerHost,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serviceErr := make(chan error, 1)
	registry := prometheus.NewRegistry()
	go func() {
		serviceErr <- visitcrawl.RunService(ctx, cfg, registry)
	}()

	consumer := ordersConsumer(t, ctx, crawlNATSURL)
	waitForListening(t, cfg.ListenAddr)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	visitReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		signedVisitURL(cfg.ListenAddr, "https://example.org/a", cfg.LinkSecret),
		nil,
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := client.Do(visitReq)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "https://example.org/a" {
		t.Fatalf("location = %q", got)
	}

	order := nextPlacedOrder(t, consumer)
	if len(order.SeedURLs) != 1 ||
		order.SeedURLs[0] != canonicalurltest.CanonicalURLOf(t, "https://example.org/a") {
		t.Fatalf("order seeds = %v", order.SeedURLs)
	}

	cancel()
	if err := <-serviceErr; err != nil {
		t.Fatalf("RunService: %v", err)
	}
}

func TestRunServiceRejectsBadCrawlNATSURL(t *testing.T) {
	cfg := visitcrawl.ServiceConfig{
		CrawlNATSURL: "nats://127.0.0.1:1", ListenAddr: "127.0.0.1:0", OpsAddr: "127.0.0.1:0",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	registry := prometheus.NewRegistry()
	err := visitcrawl.RunService(ctx, cfg, registry)
	if err == nil {
		t.Fatal("unreachable nats should fail")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected connect failure, got deadline exceeded")
	}
}

func signedVisitURL(listenAddr, visitedPage, secret string) string {
	seconds := strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(seconds + "\n" + visitedPage))
	return fmt.Sprintf("http://%s/visit?url=%s&expires=%s&signature=%s",
		listenAddr, url.QueryEscape(visitedPage), seconds, hex.EncodeToString(seal.Sum(nil)))
}

func ordersConsumer(t *testing.T, ctx context.Context, crawlNATSURL string) jetstream.Consumer {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, crawlNATSURL)
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{visitcrawl.DefaultOrdersSubject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	consumer, err := js.CreateOrUpdateConsumer(ctx, yacycrawlcontract.OrdersStreamName,
		jetstream.ConsumerConfig{
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: visitcrawl.DefaultOrdersSubject,
		})
	if err != nil {
		t.Fatal(err)
	}
	return consumer
}

func nextPlacedOrder(t *testing.T, consumer jetstream.Consumer) yacycrawlcontract.CrawlOrder {
	t.Helper()
	msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("fetch placed order: %v", err)
	}
	order, err := yacycrawlcontract.UnmarshalCrawlOrder(msg.Data())
	if err != nil {
		t.Fatalf("decode order: %v", err)
	}
	return order
}

func waitForListening(t *testing.T, addr string) {
	t.Helper()
	dialer := &net.Dialer{}
	waitFor(t, func() bool {
		conn, err := dialer.DialContext(context.Background(), "tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
