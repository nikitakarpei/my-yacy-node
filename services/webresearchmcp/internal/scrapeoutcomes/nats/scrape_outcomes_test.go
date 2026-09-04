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
	data []byte,
) {
	t.Helper()
	connection := natstestserver.Connect(t, serverURL)
	if err := connection.Publish(subject, data); err != nil {
		t.Fatalf("carry the outcome: %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("confirm the carried outcome: %v", err)
	}
}

func awaitedFetchOutcome(
	t *testing.T,
	subjectOf func(canonicalurl.CanonicalURL) string,
	dataOf func(*testing.T, canonicalurl.CanonicalURL) []byte,
) pageread.FetchOutcome {
	t.Helper()
	serverURL := natstestserver.Start(t)
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)

	outcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(
		serverURL, &recordingScrapeOutcomeObserver{},
	)
	if err != nil {
		t.Fatalf("open the scrape outcomes: %v", err)
	}
	defer outcomes.Close()

	listener, err := outcomes.ListenerFor(context.Background(), pageURL)
	if err != nil {
		t.Fatalf("listen for the outcome: %v", err)
	}
	defer listener.Close()

	carry(t, serverURL, subjectOf(pageURL), dataOf(t, pageURL))

	waitCtx, stopWaiting := context.WithTimeout(context.Background(), waitLimit)
	defer stopWaiting()
	return listener.AwaitedFetchOutcome(waitCtx)
}

func keptPageReceiptFrom(
	t *testing.T,
	pageURL canonicalurl.CanonicalURL,
) []byte {
	t.Helper()
	data, err := pagescrapecontract.MarshalKeptPage(pagescrapecontract.KeptPage{
		PageURL: pageURL,
		Corpus:  pagescrapecontract.CorpusMarkdown,
	})
	if err != nil {
		t.Fatalf("marshal kept page: %v", err)
	}
	return data
}

func rejectedPageReceiptFrom(
	t *testing.T,
	pageURL canonicalurl.CanonicalURL,
) []byte {
	t.Helper()
	data, err := pagescrapecontract.MarshalRejectedPage(pagescrapecontract.RejectedPage{
		PageURL: pageURL,
		Corpus:  pagescrapecontract.CorpusMarkdown,
	})
	if err != nil {
		t.Fatalf("marshal rejected page: %v", err)
	}
	return data
}

func scrapeFailureFrom(
	t *testing.T,
	pageURL canonicalurl.CanonicalURL,
) []byte {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeFailure(pagescrapecontract.ScrapeFailure{
		PageURL:  pageURL,
		FetchURL: pageURL,
		Reason:   pagescrapecontract.NoReasonGiven,
	})
	if err != nil {
		t.Fatalf("marshal scrape failure: %v", err)
	}
	return data
}

func TestAPageACorpusKeptIsAwaitedAsAFetchedPage(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(
		t, pagescrapecontract.KeptPageOutcomeSubjectOf, keptPageReceiptFrom,
	)

	if fetchOutcome != pageread.PageFetched {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageFetched)
	}
}

func TestAPageCorpusMarkdownRejectedIsAwaitedAsAPageThatCannotBeRead(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(
		t, pagescrapecontract.RejectedPageOutcomeSubjectOf, rejectedPageReceiptFrom,
	)

	if fetchOutcome != pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageNotReadable)
	}
}

func TestAScrapeThatFailedIsAwaitedAsAPageThatCannotBeRead(t *testing.T) {
	fetchOutcome := awaitedFetchOutcome(
		t, pagescrapecontract.ScrapeFailureOutcomeSubjectOf, scrapeFailureFrom,
	)

	if fetchOutcome != pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageNotReadable)
	}
}

func TestAReceiptFromAnotherCorpusDoesNotFinishTheWait(t *testing.T) {
	serverURL := natstestserver.Start(t)
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)
	outcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(
		serverURL, &recordingScrapeOutcomeObserver{},
	)
	if err != nil {
		t.Fatalf("open the scrape outcomes: %v", err)
	}
	defer outcomes.Close()
	listener, err := outcomes.ListenerFor(context.Background(), pageURL)
	if err != nil {
		t.Fatalf("listen for the outcome: %v", err)
	}
	defer listener.Close()

	otherReceipt, err := pagescrapecontract.MarshalKeptPage(pagescrapecontract.KeptPage{
		PageURL: pageURL,
		Corpus:  "corpustext",
	})
	if err != nil {
		t.Fatalf("marshal another corpus receipt: %v", err)
	}
	carry(t, serverURL, pagescrapecontract.KeptPageOutcomeSubjectOf(pageURL), otherReceipt)
	carry(
		t, serverURL, pagescrapecontract.RejectedPageOutcomeSubjectOf(pageURL),
		rejectedPageReceiptFrom(t, pageURL),
	)

	waitCtx, stopWaiting := context.WithTimeout(context.Background(), waitLimit)
	defer stopWaiting()
	fetchOutcome := listener.AwaitedFetchOutcome(waitCtx)
	if fetchOutcome != pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageNotReadable)
	}
}

func TestWaitThatRunsOutBeforeAnyAnnouncementIsUnfinished(t *testing.T) {
	serverURL := natstestserver.Start(t)
	outcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(
		serverURL, &recordingScrapeOutcomeObserver{},
	)
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
	if fetchOutcome := listener.AwaitedFetchOutcome(waitCtx); fetchOutcome !=
		pageread.FetchUnfinished {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.FetchUnfinished)
	}
}

func TestAMessageThatCannotBeReadIsObservedAndDoesNotFinishTheWait(t *testing.T) {
	serverURL := natstestserver.Start(t)
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)
	observer := &recordingScrapeOutcomeObserver{}
	outcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(serverURL, observer)
	if err != nil {
		t.Fatalf("open the scrape outcomes: %v", err)
	}
	defer outcomes.Close()
	listener, err := outcomes.ListenerFor(context.Background(), pageURL)
	if err != nil {
		t.Fatalf("listen for the outcome: %v", err)
	}
	defer listener.Close()

	carry(
		t, serverURL, pagescrapecontract.KeptPageOutcomeSubjectOf(pageURL),
		[]byte("not a kept page"),
	)
	carry(
		t, serverURL, pagescrapecontract.RejectedPageOutcomeSubjectOf(pageURL),
		rejectedPageReceiptFrom(t, pageURL),
	)

	waitCtx, stopWaiting := context.WithTimeout(context.Background(), waitLimit)
	defer stopWaiting()
	if fetchOutcome := listener.AwaitedFetchOutcome(waitCtx); fetchOutcome !=
		pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", fetchOutcome, pageread.PageNotReadable)
	}
	if observer.malformedMessages != 1 {
		t.Errorf("malformed messages = %d, want 1", observer.malformedMessages)
	}
}

func TestScrapeOutcomesCannotOpenOnAServerThatIsAway(t *testing.T) {
	if _, err := scrapeoutcomesnats.OpenScrapeOutcomes(
		"nats://127.0.0.1:1", &recordingScrapeOutcomeObserver{},
	); err == nil {
		t.Fatal("the scrape outcomes opened, want the server that is away to fail")
	}
}

type recordingScrapeOutcomeObserver struct {
	subscriptionFailures         int
	listenerConfirmationFailures int
	malformedMessages            int
}

func (o *recordingScrapeOutcomeObserver) ScrapeOutcomeSubscriptionFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.subscriptionFailures++
}

func (o *recordingScrapeOutcomeObserver) ScrapeOutcomeListenerConfirmationFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.listenerConfirmationFailures++
}

func (o *recordingScrapeOutcomeObserver) ScrapeOutcomeMessageMalformed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.malformedMessages++
}
