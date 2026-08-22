package main_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	firecrawlshim "github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/cmd/firecrawlshim"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release address: %v", err)
	}
	return addr
}

func testConfig(crawlNATSURL, listenAddr string) firecrawlshim.ServiceConfig {
	return firecrawlshim.ServiceConfig{
		CrawlNATSURL:         crawlNATSURL,
		OrdersSubject:        firecrawlshim.DefaultOrdersSubject,
		ListenAddr:           listenAddr,
		CrawlOutcomesTarget:  "passthrough:///crawler:8095",
		MarkdownCorpusTarget: "passthrough:///corpusmarkdown:8094",
		RecallLimit:          time.Second,
		PollInterval:         50 * time.Millisecond,
		MaxInFlight:          firecrawlshim.DefaultMaxInFlight,
	}
}

func TestRunServiceFailsWhenCrawlNATSUnreachable(t *testing.T) {
	cfg := testConfig("nats://127.0.0.1:1", freeAddr(t))

	if err := firecrawlshim.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected an error when the crawl nats is unreachable")
	}
}

func TestRunServiceFailsWhenTheCrawlOutcomesTargetIsInvalid(t *testing.T) {
	cfg := testConfig(natstestserver.Start(t), freeAddr(t))
	cfg.CrawlOutcomesTarget = "\x00"

	if err := firecrawlshim.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected an error for an invalid crawl outcomes target")
	}
}

func TestRunServiceFailsWhenTheMarkdownCorpusTargetIsInvalid(t *testing.T) {
	cfg := testConfig(natstestserver.Start(t), freeAddr(t))
	cfg.MarkdownCorpusTarget = "\x00"

	if err := firecrawlshim.RunService(context.Background(), cfg); err == nil {
		t.Fatal("expected an error for an invalid markdown corpus target")
	}
}

func TestRunServiceServesUntilContextCancelled(t *testing.T) {
	addr := freeAddr(t)
	cfg := testConfig(natstestserver.Start(t), addr)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- firecrawlshim.RunService(ctx, cfg) }()
	waitForListening(t, addr)

	if code := scrapeStatus(t, ctx, addr); code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 (unreachable crawler)", code)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("RunService = %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("RunService did not return after cancel")
	}
}

func scrapeStatus(t *testing.T, ctx context.Context, addr string) int {
	t.Helper()
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, "http://"+addr+"/v1/scrape",
		strings.NewReader(`{"url":"https://example.com"}`),
	)
	if err != nil {
		t.Fatalf("build scrape request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post scrape: %v", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	return response.StatusCode
}

func waitForListening(t *testing.T, addr string) {
	t.Helper()
	dialer := net.Dialer{Timeout: 100 * time.Millisecond}
	for range 100 {
		conn, err := dialer.DialContext(context.Background(), "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server never listened on %s", addr)
}
