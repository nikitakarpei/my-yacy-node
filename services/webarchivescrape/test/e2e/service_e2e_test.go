//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/scraperequeststream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/warcarchive"
)

const originURL = "http://origin.test/"

func TestDryRunSelectsNewestHTMLCapturesFromPywb(t *testing.T) {
	ctx := context.Background()
	network := dockernetwork.New(t, ctx)
	archive := startPywbArchive(t, ctx, network.Name, []warcarchive.Capture{
		{URL: originURL, CapturedAt: moment(1), Body: htmlPage("old root")},
		{URL: originURL, CapturedAt: moment(2), Body: htmlPage("new root")},
		{URL: originURL + "other", CapturedAt: moment(1), Body: htmlPage("other")},
		{
			URL:         originURL + "paper.pdf",
			CapturedAt:  moment(1),
			ContentType: "application/pdf",
			Body:        "pdf",
		},
		{
			URL:        originURL + "missing",
			CapturedAt: moment(1),
			StatusCode: 404,
			Body:       htmlPage("missing"),
		},
	})

	output := runWebArchiveScrape(
		t,
		ctx,
		network.Name,
		commandArguments(archive.NetworkURL(), originURL, true),
		nil,
	)
	replayAddresses := replayAddressesFrom(output)
	want := []string{
		archive.NetworkURL() + "/captures/20240201000000mp_/" + originURL,
		archive.NetworkURL() + "/captures/20240101000000mp_/" + originURL + "other",
	}
	if strings.Join(replayAddresses, "\n") != strings.Join(want, "\n") {
		t.Fatalf(
			"dry-run replay addresses = %q, want %q; complete output: %s",
			replayAddresses,
			want,
			output,
		)
	}
}

func TestPublishesTheCapturedAddressAndTheReplayAddressFromPywb(t *testing.T) {
	ctx := context.Background()
	network := dockernetwork.New(t, ctx)
	archive := startPywbArchive(
		t,
		ctx,
		network.Name,
		[]warcarchive.Capture{
			{URL: originURL, CapturedAt: moment(1), Body: htmlPage("published")},
		},
	)
	natsURL := natsjetstream.Start(t, ctx, network.Name)
	scraperequeststream.Provision(t, ctx, natsURL)
	subscription := subscribeToScrapeRequests(t, natsURL)

	runWebArchiveScrape(
		t,
		ctx,
		network.Name,
		commandArguments(archive.NetworkURL(), originURL, false),
		map[string]string{"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL()},
	)
	wantReplayAddress := archive.NetworkURL() + "/captures/20240101000000mp_/" + originURL
	published := subscription.next(t)
	if got := published.PageURL.String(); got != originURL {
		t.Fatalf("published page address = %q, want the captured address %q", got, originURL)
	}
	if got := published.FetchURL.String(); got != wantReplayAddress {
		t.Fatalf(
			"published fetch address = %q, want the replay address %q",
			got,
			wantReplayAddress,
		)
	}
}

func TestCorpusTextIndexesCapturedAddressFromPywbReplay(t *testing.T) {
	ctx := context.Background()
	network := dockernetwork.New(t, ctx)
	archive := startPywbArchive(t, ctx, network.Name, []warcarchive.Capture{
		{URL: originURL, CapturedAt: moment(1), Body: htmlPage("discarded archive words")},
		{
			URL:        originURL,
			CapturedAt: moment(2),
			Body:       htmlPage("distinct newest archive words"),
		},
	})
	natsURL := natsjetstream.Start(t, ctx, network.Name)
	scraperequeststream.Provision(t, ctx, natsURL)
	egressproxy.Start(t, ctx, network.Name)
	manticoreURL := manticore.Start(t, ctx, network.Name, "manticore")
	startCorpusText(t, ctx, network.Name)

	runWebArchiveScrape(
		t,
		ctx,
		network.Name,
		commandArguments(archive.NetworkURL(), originURL, false),
		map[string]string{"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL()},
	)
	page := indexedPageContaining(t, ctx, manticoreURL, "distinct newest archive words")
	if page.URL != originURL {
		t.Fatalf("indexed address = %q, want captured address %q", page.URL, originURL)
	}
	if strings.Contains(page.URL, archive.NetworkURL()) {
		t.Fatalf("indexed address %q uses replay address", page.URL)
	}
}

func TestAbsentDomainPublishesNothing(t *testing.T) {
	ctx := context.Background()
	network := dockernetwork.New(t, ctx)
	archive := startPywbArchive(
		t,
		ctx,
		network.Name,
		[]warcarchive.Capture{
			{URL: originURL, CapturedAt: moment(1), Body: htmlPage("present")},
		},
	)
	natsURL := natsjetstream.Start(t, ctx, network.Name)
	scraperequeststream.Provision(t, ctx, natsURL)
	subscription := subscribeToScrapeRequests(t, natsURL)

	runWebArchiveScrape(
		t,
		ctx,
		network.Name,
		commandArguments(archive.NetworkURL(), "http://absent.test/", false),
		map[string]string{"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL()},
	)
	if subscription.hasMessage() {
		t.Fatal("absent domain published a scrape request")
	}
}

func moment(month time.Month) time.Time {
	return time.Date(2024, month, 1, 0, 0, 0, 0, time.UTC)
}

func htmlPage(content string) string {
	return fmt.Sprintf(
		"<!doctype html><html lang=\"en\"><title>Archive</title><body>%s</body></html>",
		content,
	)
}

func replayAddressesFrom(output string) []string {
	var addresses []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "http://pywb:8080/") {
			addresses = append(addresses, line)
		}
	}
	return addresses
}
