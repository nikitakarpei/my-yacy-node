package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	scrapeoutcomesnats "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomes/nats"
)

const (
	pageAddress = "https://example.org/page"
	waitLimit   = 5 * time.Second
)

func announce(
	t *testing.T,
	serverURL string,
	pageURL canonicalurl.CanonicalURL,
	outcome pagemarkdownstore.ScrapeOutcome,
) {
	t.Helper()
	notice, err := pagemarkdownstore.MarshalScrapeOutcomeNotice(
		pagemarkdownstore.ScrapeOutcomeNotice{RequestedURL: pageURL, Outcome: outcome},
	)
	if err != nil {
		t.Fatalf("marshal the notice: %v", err)
	}
	connection := natstestserver.Connect(t, serverURL)
	if err := connection.Publish(
		pagemarkdownstore.ScrapeOutcomeSubjectOf(pageURL),
		notice,
	); err != nil {
		t.Fatalf("announce the outcome: %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("confirm the announcement: %v", err)
	}
}

func awaitedFetchOutcome(
	t *testing.T,
	outcome pagemarkdownstore.ScrapeOutcome,
) pageread.FetchOutcome {
	t.Helper()
	serverURL := natstestserver.Start(t)
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	outcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(serverURL)
	if err != nil {
		t.Fatalf("open the scrape outcomes: %v", err)
	}
	defer outcomes.Close()

	listener, err := outcomes.ListenerFor(context.Background(), pageURL)
	if err != nil {
		t.Fatalf("listen for the outcome: %v", err)
	}
	defer listener.Close()

	announce(t, serverURL, pageURL, outcome)

	waitCtx, stopWaiting := context.WithTimeout(context.Background(), waitLimit)
	defer stopWaiting()
	fetchOutcome, err := listener.AwaitedFetchOutcome(waitCtx)
	if err != nil {
		t.Fatalf("wait for the outcome: %v", err)
	}
	return fetchOutcome
}

func TestStoredMarkdownIsAwaitedAsAFetchedPage(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(t, pagemarkdownstore.MarkdownStored)

	if fetchOutcome != pageread.PageFetched {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageFetched)
	}
}

func TestAGivenUpPageIsAwaitedAsAPageThatCannotBeRead(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(t, pagemarkdownstore.PageGivenUp)

	if fetchOutcome != pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageNotReadable)
	}
}

func TestWaitThatRunsOutBeforeAnyAnnouncementFails(t *testing.T) {
	serverURL := natstestserver.Start(t)
	outcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(serverURL)
	if err != nil {
		t.Fatalf("open the scrape outcomes: %v", err)
	}
	defer outcomes.Close()

	listener, err := outcomes.ListenerFor(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	)
	if err != nil {
		t.Fatalf("listen for the outcome: %v", err)
	}
	defer listener.Close()

	waitCtx, stopWaiting := context.WithTimeout(context.Background(), time.Millisecond)
	defer stopWaiting()
	if _, err := listener.AwaitedFetchOutcome(waitCtx); err == nil {
		t.Fatal("the wait answered without an error, want the wait that ran out to fail")
	}
}

func TestScrapeOutcomesCannotOpenOnAServerThatIsAway(t *testing.T) {
	if _, err := scrapeoutcomesnats.OpenScrapeOutcomes("nats://127.0.0.1:1"); err == nil {
		t.Fatal("the scrape outcomes opened, want the server that is away to fail")
	}
}
