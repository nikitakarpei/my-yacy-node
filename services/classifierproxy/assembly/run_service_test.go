package classifierproxyassembly_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/pagefeatures"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/safetensorsmodel"
	classifierproxy "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const pageBody = "<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>"

func TestTheServiceServesTheClassificationByTheModelOfEachCategory(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, pageBody)
	}))
	defer egress.Close()
	cfg := serviceConfigFor(t, egress.URL, map[pageclassification.Category][]byte{
		pageclassification.Spam: safetensorsmodel.SafetensorsFrom(spamModel()),
		"adult": safetensorsmodel.SafetensorsFrom(
			classifiermodel.Model{Version: "2026-11", Threshold: 0.8, Intercept: -5},
		),
	})
	listeners := classifierproxy.Listeners{Proxy: listenerFor(t)}
	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	go func() {
		serviceDone <- classifierproxy.RunService(ctx, cfg, listeners, prometheus.NewRegistry())
	}()

	classification := proxiedClassificationOf(
		t, listeners.Proxy.Addr().String(), "http://site.example/page",
	)
	cancel()

	if want := `spam;score=0.993;threshold=0.800;model="2026-09", ` +
		`adult=?0;score=0.007;threshold=0.800;model="2026-11"`; classification != want {
		t.Fatalf("classification %q, want %q", classification, want)
	}
	if err := <-serviceDone; err != nil {
		t.Fatalf("RunService: %v", err)
	}
}

func TestTheServiceDoesNotStartWithAModelFileOfAnotherRecipe(t *testing.T) {
	modelFile := bytes.Replace(safetensorsmodel.SafetensorsFrom(spamModel()),
		fmt.Appendf(nil, `"recipeVersion":"%d"`, pagefeatures.RecipeVersion),
		fmt.Appendf(nil, `"recipeVersion":"%d"`, pagefeatures.RecipeVersion+1), 1)
	cfg := serviceConfigFor(
		t, "http://127.0.0.1:1", map[pageclassification.Category][]byte{"spam": modelFile},
	)
	listeners := classifierproxy.Listeners{Proxy: listenerFor(t)}

	err := classifierproxy.RunService(t.Context(), cfg, listeners, prometheus.NewRegistry())

	var mismatch safetensorsmodel.RecipeVersionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("RunService: %v", err)
	}
}

func TestTheServiceDoesNotStartWithoutItsModelFile(t *testing.T) {
	cfg := serviceConfigFor(t, "http://127.0.0.1:1", nil)
	cfg.SafetensorsModelPathPerCategory = map[pageclassification.Category]string{
		pageclassification.Spam: filepath.Join(t.TempDir(), "absent.safetensors"),
	}
	listeners := classifierproxy.Listeners{Proxy: listenerFor(t)}

	if err := classifierproxy.RunService(
		t.Context(),
		cfg,
		listeners,
		prometheus.NewRegistry(),
	); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf("RunService: %v", err)
	}
}

func TestListenersForListensOnTheProxyAddress(t *testing.T) {
	listeners, err := classifierproxy.ListenersFor(
		t.Context(), classifierproxy.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Proxy.Close() })
	if listeners.Proxy.Addr().String() == "127.0.0.1:0" {
		t.Errorf("proxy listener = %s, want a bound port", listeners.Proxy.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := classifierproxy.ListenersFor(
		t.Context(), classifierproxy.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable proxy address")
	}
}

func spamModel() classifiermodel.Model {
	return classifiermodel.Model{Version: "2026-09", Threshold: 0.8, Intercept: 5}
}

func serviceConfigFor(
	t *testing.T,
	egressURL string,
	modelFilePerCategory map[pageclassification.Category][]byte,
) classifierproxy.ServiceConfig {
	t.Helper()
	safetensorsModelPathPerCategory := map[pageclassification.Category]string{}
	for category, modelFile := range modelFilePerCategory {
		safetensorsModelPath := filepath.Join(t.TempDir(), string(category)+".safetensors")
		if err := os.WriteFile(safetensorsModelPath, modelFile, 0o600); err != nil {
			t.Fatal(err)
		}
		safetensorsModelPathPerCategory[category] = safetensorsModelPath
	}
	egressProxyURL, err := url.Parse(egressURL)
	if err != nil {
		t.Fatal(err)
	}
	return classifierproxy.ServiceConfig{
		EgressProxyURL:                  egressProxyURL,
		EgressProxyDialMode:             classifierproxy.DialModeTunnel,
		SafetensorsModelPathPerCategory: safetensorsModelPathPerCategory,
		PageByteCeiling:                 1000,
		MaxPagesReadAtOnce:              4,
		ResponseHeaderTimeout:           5 * time.Second,
		RelayIdleTimeout:                5 * time.Second,
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

func proxiedClassificationOf(t *testing.T, proxyAddress, address string) string {
	t.Helper()
	proxyURL := &url.URL{Scheme: "http", Host: proxyAddress}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	return response.Header.Get("Page-Classification")
}
