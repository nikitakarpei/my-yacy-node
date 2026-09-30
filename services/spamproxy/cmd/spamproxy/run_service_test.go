package main_test

import (
	"bytes"
	"context"
	"errors"
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

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/safetensorsmodel"
	spamproxy "github.com/nikitakarpei/yacy-rwi-node/spamproxy/cmd/spamproxy"
)

const pageBody = "<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>"

func TestTheServiceServesTheVerdictsOfItsModelFile(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, pageBody)
	}))
	defer egress.Close()
	cfg := serviceConfigFor(t, egress.URL, safetensorsmodel.SafetensorsFrom(spamModel()))
	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	go func() { serviceDone <- spamproxy.RunService(ctx, cfg, prometheus.NewRegistry()) }()

	verdict := proxiedVerdictOf(t, cfg.ListenAddr, "http://site.example/page")
	cancel()

	if !strings.HasPrefix(verdict, "spam;") {
		t.Fatalf("verdict %q", verdict)
	}
	if err := <-serviceDone; err != nil {
		t.Fatalf("RunService: %v", err)
	}
}

func TestTheServiceDoesNotStartWithAModelFileOfAnotherRecipe(t *testing.T) {
	modelFile := bytes.Replace(safetensorsmodel.SafetensorsFrom(spamModel()),
		[]byte(`"recipeVersion":"5"`), []byte(`"recipeVersion":"6"`), 1)
	cfg := serviceConfigFor(t, "http://127.0.0.1:1", modelFile)

	err := spamproxy.RunService(t.Context(), cfg, prometheus.NewRegistry())

	var mismatch safetensorsmodel.RecipeVersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("RunService: %v", err)
	}
}

func TestTheServiceDoesNotStartWithoutItsModelFile(t *testing.T) {
	cfg := serviceConfigFor(t, "http://127.0.0.1:1", nil)
	cfg.SafetensorsModelPath = filepath.Join(t.TempDir(), "absent.safetensors")

	if err := spamproxy.RunService(
		t.Context(),
		cfg,
		prometheus.NewRegistry(),
	); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf("RunService: %v", err)
	}
}

func spamModel() spammodel.Model {
	return spammodel.Model{Version: "2026-09", Threshold: 0.8, Intercept: 5}
}

func serviceConfigFor(t *testing.T, egressURL string, modelFile []byte) spamproxy.ServiceConfig {
	t.Helper()
	safetensorsModelPath := filepath.Join(t.TempDir(), "spam-model.safetensors")
	if err := os.WriteFile(safetensorsModelPath, modelFile, 0o600); err != nil {
		t.Fatal(err)
	}
	egressProxyURL, err := url.Parse(egressURL)
	if err != nil {
		t.Fatal(err)
	}
	return spamproxy.ServiceConfig{
		ListenAddr:            reservedAddress(t),
		EgressProxyURL:        egressProxyURL,
		EgressProxyDialMode:   spamproxy.DialModeTunnel,
		SafetensorsModelPath:  safetensorsModelPath,
		PageByteCeiling:       1000,
		MaxPagesReadAtOnce:    4,
		ResponseHeaderTimeout: 5 * time.Second,
		RelayIdleTimeout:      5 * time.Second,
		OpsAddr:               reservedAddress(t),
	}
}

func reservedAddress(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func proxiedVerdictOf(t *testing.T, proxyAddress, address string) string {
	t.Helper()
	proxyURL := &url.URL{Scheme: "http", Host: proxyAddress}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	for {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			return response.Header.Get("Spam-Assessment")
		}
		if t.Context().Err() != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
