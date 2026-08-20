//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxngsearch"
)

func TestResultLinkRouterRoutesSearchResultsThroughVisitcrawl(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	startSearchUpstream(t, ctx, network.Name)
	startVisitcrawl(t, ctx, network.Name)
	startSearXNG(t, ctx, network.Name)
	resultsOriginURL := startResultsOrigin(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createOrdersStream(t, ctx, js)

	result := searxngsearch.OneResultInAnyLanguage(
		t, ctx, resultsOriginURL, "!"+testEngineBang+" test",
	)

	wantPrefix := visitPath + "?"
	if !strings.HasPrefix(result.URL, wantPrefix) {
		t.Fatalf("result url = %q, want prefix %q", result.URL, wantPrefix)
	}
	routedLink, err := url.Parse(result.URL)
	if err != nil {
		t.Fatalf("parse routed url: %v", err)
	}
	routedParams := routedLink.Query()
	if gotDestination := routedParams.Get("url"); gotDestination != destinationPageURL {
		t.Fatalf("routed destination = %q, want %q", gotDestination, destinationPageURL)
	}
	if routedParams.Get("expires") == "" || routedParams.Get("signature") == "" {
		t.Fatalf("routed url = %q, want expires and signature", result.URL)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, resultsOriginURL+result.URL, nil,
	)
	if err != nil {
		t.Fatalf("build visit request: %v", err)
	}
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("visit request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("visit status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	if location := resp.Header.Get("Location"); location != destinationPageURL {
		t.Fatalf("visit redirect location = %q, want %q", location, destinationPageURL)
	}

	order := fetchOnePlacedOrder(t, ctx, js)
	if len(order.SeedURLs) != 1 || order.SeedURLs[0] != destinationPageURL {
		t.Fatalf("placed order seed urls = %v, want [%q]", order.SeedURLs, destinationPageURL)
	}
}
