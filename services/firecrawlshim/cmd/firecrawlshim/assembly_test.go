package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
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

func TestRunServiceRejectsInvalidTarget(t *testing.T) {
	err := RunService(context.Background(), ServiceConfig{
		ListenAddr:    freeAddr(t),
		RecallTarget:  "\x00",
		RecallTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected error for invalid recall target")
	}
}

func TestRunServiceServesUntilContextCancelled(t *testing.T) {
	addr := freeAddr(t)
	cfg := ServiceConfig{
		ListenAddr:    addr,
		RecallTarget:  "passthrough:///corpusrecall:8092",
		RecallTimeout: time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunService(ctx, cfg) }()

	waitForListening(t, addr)

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, "http://"+addr+"/v1/scrape",
		strings.NewReader(`{"url":"https://example.com"}`),
	)
	if err != nil {
		t.Fatalf("build scrape request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post scrape: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 (unreachable recall)", resp.StatusCode)
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
