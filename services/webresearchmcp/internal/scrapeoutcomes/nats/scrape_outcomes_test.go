package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	scrapeoutcomesnats "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomes/nats"
)

const (
	pageAddress = "https://example.org/page"
	waitLimit   = 5 * time.Second
)

func carry(
	t *testing.T,
	serverURL string,
	subject string,
) {
	t.Helper()
	connection := natstestserver.Connect(t, serverURL)
	if err := connection.Publish(subject, []byte("{}")); err != nil {
		t.Fatalf("carry the outcome: %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("confirm the carried outcome: %v", err)
	}
}

func awaitedFetchOutcome(
	t *testing.T,
	subjectOf func(canonicalurl.CanonicalURL) string,
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

	carry(t, serverURL, subjectOf(pageURL))

	waitCtx, stopWaiting := context.WithTimeout(context.Background(), waitLimit)
	defer stopWaiting()
	fetchOutcome, err := listener.AwaitedFetchOutcome(waitCtx)
	if err != nil {
		t.Fatalf("wait for the outcome: %v", err)
	}
	return fetchOutcome
}

func TestAPageACorpusKeptIsAwaitedAsAFetchedPage(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(t, pagescrapecontract.KeptPageOutcomeSubjectOf)

	if fetchOutcome != pageread.PageFetched {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageFetched)
	}
}

func TestAPageEveryCorpusRejectedIsAwaitedAsAPageThatCannotBeRead(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(t, pagescrapecontract.RejectedPageOutcomeSubjectOf)

	if fetchOutcome != pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageNotReadable)
	}
}

func TestAScrapeThatFailedIsAwaitedAsAPageThatCannotBeRead(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(t, pagescrapecontract.ScrapeFailureOutcomeSubjectOf)

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
