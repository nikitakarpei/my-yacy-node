//go:build e2e

// Package corpustextservice starts the service that indexes the text of offered pages and
// answers searches of it. Start returns once the service registered its page offer
// consumer, so a suite may scrape at once without losing an offer.
package corpustextservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

const (
	alias             = "corpustext"
	searchPortNumber  = "8096"
	searchPort        = searchPortNumber + "/tcp"
	opsPort           = "9090/tcp"
	envImage          = "CORPUSTEXT_IMAGE"
	durable           = "corpustext"
	durableAppearance = 60 * time.Second
	matchAppearance   = 60 * time.Second
	stopTimeout       = 30 * time.Second
)

type CorpusText struct {
	container testcontainers.Container
}

func Start(
	t *testing.T,
	ctx context.Context,
	networkName, natsURL string,
	env map[string]string,
) CorpusText {
	t.Helper()
	serviceEnv := map[string]string{
		"SCRAPE_PAGE_OFFER_NATS_URL": natsjetstream.NetworkURL(),
		"LOG_LEVEL":                  "debug",
	}
	for key, value := range env {
		serviceEnv[key] = value
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envImage, "corpustext", "e2e"),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			ExposedPorts:   []string{opsPort, searchPort},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort(opsPort),
				wait.ForListeningPort(searchPort),
			),
			Env: serviceEnv,
		},
	})
	if err != nil {
		t.Fatalf("start corpustext container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpustext", container)
	awaitDurable(t, ctx, natsURL)
	return CorpusText{container: container}
}

func awaitDurable(t *testing.T, ctx context.Context, natsURL string) {
	t.Helper()
	connection, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	defer connection.Close()
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	registered := pollwait.For(durableAppearance, func() bool {
		stream, err := js.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
		if err != nil {
			return false
		}
		_, err = stream.Consumer(ctx, durable)
		return err == nil
	})
	if !registered {
		t.Fatalf("corpustext did not register its %q durable within %s",
			durable, durableAppearance)
	}
}

func NetworkSearchURL() string {
	return "http://" + alias + ":" + searchPortNumber
}

func (corpus CorpusText) Restart(t *testing.T, ctx context.Context) {
	t.Helper()
	timeout := stopTimeout
	if err := corpus.container.Stop(ctx, &timeout); err != nil {
		t.Fatalf("stop corpustext container: %v", err)
	}
	if err := corpus.container.Start(ctx); err != nil {
		t.Fatalf("restart corpustext container: %v", err)
	}
}

func (corpus CorpusText) FirstMatchFor(
	t *testing.T,
	ctx context.Context,
	query, language string,
) searchdocument.PageMatch {
	t.Helper()
	var firstMatch searchdocument.PageMatch
	found := pollwait.For(matchAppearance, func() bool {
		matches := corpus.MatchesFor(t, ctx, query, language)
		if len(matches) == 0 {
			return false
		}
		firstMatch = matches[0]
		return true
	})
	if !found {
		t.Fatalf("corpustext never matched %q within %s", query, matchAppearance)
	}
	return firstMatch
}

func (corpus CorpusText) MatchesFor(
	t *testing.T,
	ctx context.Context,
	query, language string,
) []searchdocument.PageMatch {
	t.Helper()
	search := url.Values{searchdocument.QueryParam: {query}}
	if language != "" {
		search.Set(searchdocument.LanguageParam, language)
	}
	target := corpus.searchURL(t, ctx) + searchdocument.SearchPath + "?" + search.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("build search request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	var pageMatches searchdocument.PageMatches
	if resp.StatusCode != http.StatusOK ||
		json.NewDecoder(resp.Body).Decode(&pageMatches) != nil {
		return nil
	}
	return pageMatches.Matches
}

func (corpus CorpusText) searchURL(t *testing.T, ctx context.Context) string {
	t.Helper()
	endpoint, err := corpus.container.PortEndpoint(ctx, searchPort, "http")
	if err != nil {
		t.Fatalf("corpustext search endpoint: %v", err)
	}
	return endpoint
}
