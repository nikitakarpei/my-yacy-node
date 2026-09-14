//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxngsearch"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
)

const (
	holderAlias = "yacy-holder-e2e"
	holderToken = "yacydhtsearchholderprobe"

	cacheProbeAlias = "yacy-holder-ranking-cache-e2e"
	cacheProbeToken = "yacydhtsearchrankingcacheprobe"

	firstWordHolderAlias  = "yacy-first-word-holder-e2e"
	secondWordHolderAlias = "yacy-second-word-holder-e2e"
	firstWordToken        = "yacydhtsearchfirstwordprobe"
	secondWordToken       = "yacydhtsearchsecondwordprobe"
	crossPeerSearchAlias  = "yacydhtsearch-cross-peer"

	askingSearchAlias  = "yacydhtsearch-asking"
	readingSearchAlias = "yacydhtsearch-reading"

	answerTimeout = 180 * time.Second
)

func TestSearXNGFindsAPeerDocumentThroughYacydhtsearch(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)

	_, holderURL := yacypeer.Start(
		t, ctx, probe, network.Name, holderAlias, yacypeer.RemoteSearchOverrides()...,
	)
	documentAddress := yacypeer.PushDocument(t, ctx, probe, holderURL, []string{holderToken})

	startYacydhtsearch(
		t, ctx, network.Name, yacydhtsearchAlias, seedlistURLOf(holderAlias), nil,
	)
	searxngURL := startSearXNG(t, ctx, network.Name, yacydhtsearchAlias)

	var found searxngsearch.Result
	answered := pollwait.For(answerTimeout, func() bool {
		for _, result := range searxngsearch.ResultsInAnyLanguage(t, ctx, searxngURL, holderToken) {
			if result.URL == documentAddress {
				found = result
				return true
			}
		}
		return false
	})
	if !answered {
		t.Fatalf(
			"SearXNG never returned %s for %q; the peer yacydhtsearch was seeded with holds it",
			documentAddress,
			holderToken,
		)
	}
	if found.Title == "" {
		t.Fatalf("SearXNG read no title from %+v", found)
	}
}

func TestASecondServiceAnswersFromTheRankingHeldInNATS(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)
	natsjetstream.Start(t, ctx, network.Name)

	_, holderURL := yacypeer.Start(
		t, ctx, probe, network.Name, cacheProbeAlias, yacypeer.RemoteSearchOverrides()...,
	)
	documentAddress := yacypeer.PushDocument(t, ctx, probe, holderURL, []string{cacheProbeToken})

	rankingCacheSettings := map[string]string{"YACYDHTSEARCH_NATS_URL": natsjetstream.NetworkURL()}
	asking := startYacydhtsearch(
		t,
		ctx,
		network.Name,
		askingSearchAlias,
		seedlistURLOf(cacheProbeAlias),
		rankingCacheSettings,
	)
	reading := startYacydhtsearch(
		t,
		ctx,
		network.Name,
		readingSearchAlias,
		seedlistURLOf(cacheProbeAlias),
		rankingCacheSettings,
	)

	found := pollwait.For(answerTimeout, func() bool {
		for _, link := range resultLinksFor(t, ctx, probe, asking.searchURL, cacheProbeToken) {
			if link == documentAddress {
				return true
			}
		}
		return false
	})
	if !found {
		t.Fatalf("%s never returned %s for %q", askingSearchAlias, documentAddress, cacheProbeToken)
	}

	links := resultLinksFor(t, ctx, probe, reading.searchURL, cacheProbeToken)
	if len(links) != 1 || links[0] != documentAddress {
		t.Fatalf("%s returned %v, want the cached %s", readingSearchAlias, links, documentAddress)
	}

	metrics := publishedBy(t, ctx, probe, reading)
	if !strings.Contains(metrics, `yacydhtsearch_searches_total{outcome="answered_from_cache"} 1`) {
		t.Fatalf("%s did not answer from the ranking cache:\n%s", readingSearchAlias, metrics)
	}
	if strings.Contains(metrics, `yacydhtsearch_searches_total{outcome="answered_by_peers"}`) {
		t.Fatalf("%s asked peers for a query it could read:\n%s", readingSearchAlias, metrics)
	}
}

func TestTwoPeersThatHoldOneQueryWordEachStillAnswerWithTheDocuments(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)
	egressproxy.Start(t, ctx, network.Name)

	_, firstWordHolderURL := yacypeer.Start(
		t, ctx, probe, network.Name, firstWordHolderAlias, yacypeer.RemoteSearchOverrides()...,
	)
	_, secondWordHolderURL := yacypeer.Start(
		t, ctx, probe, network.Name, secondWordHolderAlias, yacypeer.RemoteSearchOverrides()...,
	)
	for _, document := range firstWordDocuments() {
		yacypeer.PushDocumentUnderAddress(
			t, ctx, probe, firstWordHolderURL, document, []string{firstWordToken},
		)
	}
	for _, document := range secondWordDocuments() {
		yacypeer.PushDocumentUnderAddress(
			t, ctx, probe, secondWordHolderURL, document, []string{secondWordToken},
		)
	}

	service := startYacydhtsearch(
		t,
		ctx,
		network.Name,
		crossPeerSearchAlias,
		seedlistURLOf(firstWordHolderAlias)+","+seedlistURLOf(secondWordHolderAlias),
		map[string]string{
			"YACYDHTSEARCH_RANKING_LIFETIME": "5s",
		},
	)

	query := firstWordToken + " " + secondWordToken
	wanted := documentsOfBothWords()
	var links []string
	answered := pollwait.For(answerTimeout, func() bool {
		links = resultLinksFor(t, ctx, probe, service.searchURL, query)

		return containsEach(links, wanted)
	})
	if !answered {
		t.Fatalf(
			"%s returned %v for %q, want every document both words are held for: %v",
			crossPeerSearchAlias,
			links,
			query,
			wanted,
		)
	}
	for _, link := range links {
		if !slices.Contains(wanted, link) {
			t.Fatalf(
				"%s returned %s, which only one peer holds a query word for; all links %v",
				crossPeerSearchAlias,
				link,
				links,
			)
		}
	}
}

func firstWordDocuments() []string {
	return []string{
		crossPeerDocumentAt("both-words-one.html"),
		crossPeerDocumentAt("both-words-two.html"),
		crossPeerDocumentAt("first-word-only.html"),
	}
}

func secondWordDocuments() []string {
	return []string{
		crossPeerDocumentAt("both-words-one.html"),
		crossPeerDocumentAt("both-words-two.html"),
		crossPeerDocumentAt("second-word-only.html"),
	}
}

func documentsOfBothWords() []string {
	return []string{
		crossPeerDocumentAt("both-words-one.html"),
		crossPeerDocumentAt("both-words-two.html"),
	}
}

func crossPeerDocumentAt(path string) string {
	return "http://transfer.example.invalid/" + path
}

func containsEach(links, wanted []string) bool {
	for _, want := range wanted {
		if !slices.Contains(links, want) {
			return false
		}
	}

	return true
}

func seedlistURLOf(alias string) string {
	return "http://" + alias + ":" + peerclient.Port + "/yacy/seedlist.html"
}

func resultLinksFor(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	searchURL, query string,
) []string {
	t.Helper()

	result := probe.Get(ctx, searchURL+"/yacysearch.json?"+url.Values{
		"query":          {query},
		"maximumRecords": {"10"},
		"startRecord":    {"0"},
	}.Encode())
	if !result.OK {
		return nil
	}

	var page struct {
		Channels []struct {
			Items []struct {
				Link string `json:"link"`
			} `json:"items"`
		} `json:"channels"`
	}
	if err := json.Unmarshal([]byte(result.Body), &page); err != nil {
		t.Fatalf("parse /yacysearch.json: %v (body %q)", err, result.Body)
	}
	if len(page.Channels) == 0 {
		return nil
	}

	links := make([]string, 0, len(page.Channels[0].Items))
	for _, item := range page.Channels[0].Items {
		links = append(links, item.Link)
	}

	return links
}

func publishedBy(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	service yacydhtsearchService,
) string {
	t.Helper()

	result := probe.Get(ctx, service.opsURL+"/metrics")
	if !result.OK {
		t.Fatalf("read metrics: %s", result.Diag())
	}

	return result.Body
}
