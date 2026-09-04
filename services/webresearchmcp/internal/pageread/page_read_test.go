package pageread_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

const (
	pageAddress     = "https://example.org/page"
	storedMarkdown  = "# Research subject"
	storedVersion   = "version-1"
	fetchedVersion  = "version-2"
	characterLimit  = 8
	scrapeTolerance = time.Hour
	fetchWait       = time.Second
)

var errCorpusAway = errors.New("corpus away")

type corpusHolding struct {
	stored  []pageread.PageMarkdown
	failure error
	recalls int
}

func (c *corpusHolding) PageMarkdownAt(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) (pageread.PageMarkdown, error) {
	if c.failure != nil {
		return pageread.PageMarkdown{}, c.failure
	}
	held := c.stored[min(c.recalls, len(c.stored)-1)]
	c.recalls++
	if held.Version == "" {
		return pageread.PageMarkdown{}, pageread.ErrPageNotInCorpus
	}
	return held, nil
}

type recordedScrapeRequests struct{ asked int }

func (r *recordedScrapeRequests) AskToScrape(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
	r.asked++
}

type outcomesAnnouncing struct {
	outcome     pageread.FetchOutcome
	listenError error
	closed      bool
}

func (o *outcomesAnnouncing) ListenerFor(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) (pageread.ScrapeOutcomeListener, error) {
	if o.listenError != nil {
		return nil, o.listenError
	}
	return o, nil
}

func (o *outcomesAnnouncing) AwaitedFetchOutcome(
	_ context.Context,
) pageread.FetchOutcome {
	return o.outcome
}

func (o *outcomesAnnouncing) Close() { o.closed = true }

type recordingPageReadProgress struct {
	answeredOutcomes []pageread.FetchOutcome
	recallFailures   int
	outcomesNotHeard int
}

func (p *recordingPageReadProgress) PageAnswered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	fetchOutcome pageread.FetchOutcome,
) {
	p.answeredOutcomes = append(p.answeredOutcomes, fetchOutcome)
}

func (p *recordingPageReadProgress) MarkdownRecallFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	p.recallFailures++
}

func (p *recordingPageReadProgress) FetchOutcomeNotHeard(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	p.outcomesNotHeard++
}

type readerUnderTest struct {
	reader   *pageread.PageReader
	corpus   *corpusHolding
	requests *recordedScrapeRequests
	outcomes *outcomesAnnouncing
	progress *recordingPageReadProgress
}

func newReaderUnderTest(
	corpus *corpusHolding,
	outcomes *outcomesAnnouncing,
) readerUnderTest {
	requests := &recordedScrapeRequests{}
	progress := &recordingPageReadProgress{}
	return readerUnderTest{
		reader: pageread.NewPageReader(pageread.Config{
			Corpus:          corpus,
			ScrapeRequests:  requests,
			ScrapeOutcomes:  outcomes,
			Progress:        progress,
			CharacterLimit:  characterLimit,
			ScrapeTolerance: scrapeTolerance,
			FetchWait:       fetchWait,
		}),
		corpus:   corpus,
		requests: requests,
		outcomes: outcomes,
		progress: progress,
	}
}

func pageURLUnderTest(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	pageURL, err := canonicalurl.CanonicalURLOf(pageAddress)
	if err != nil {
		t.Fatalf("read the page address: %v", err)
	}
	return pageURL
}

func markdownStoredAt(storedAt time.Time, version string) pageread.PageMarkdown {
	return pageread.PageMarkdown{
		Markdown: storedMarkdown,
		Version:  version,
		StoredAt: storedAt,
	}
}

func TestPageNotInTheCorpusIsFetchedAndAnsweredWithWhatTheFetchStored(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			{},
			markdownStoredAt(time.Now(), fetchedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.PageFetched {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.PageFetched)
	}
	if page.Version != fetchedVersion {
		t.Errorf("version = %q, want %q", page.Version, fetchedVersion)
	}
	if under.requests.asked != 1 {
		t.Errorf("scrape requests = %d, want 1", under.requests.asked)
	}
	if !under.outcomes.closed {
		t.Error("the scrape outcome listener is left open")
	}
}

func TestMarkdownWithinTheToleratedAgeIsAnsweredWithoutAskingForAScrape(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			markdownStoredAt(time.Now(), storedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.FetchNotNeeded {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.FetchNotNeeded)
	}
	if under.requests.asked != 0 {
		t.Errorf("scrape requests = %d, want none", under.requests.asked)
	}
}

func TestMarkdownOlderThanTheToleratedAgeIsFetchedAgain(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			markdownStoredAt(time.Now().Add(-2*scrapeTolerance), storedVersion),
			markdownStoredAt(time.Now(), fetchedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.Version != fetchedVersion {
		t.Errorf("version = %q, want the fetched %q", page.Version, fetchedVersion)
	}
	if under.requests.asked != 1 {
		t.Errorf("scrape requests = %d, want 1", under.requests.asked)
	}
}

func TestCallNamingALargerAgeAcceptsMarkdownTheConfiguredAgeRejects(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			markdownStoredAt(time.Now().Add(-2*scrapeTolerance), storedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(context.Background(), pageread.PageCall{
		URL:          pageURLUnderTest(t),
		ToleratedAge: 3 * scrapeTolerance,
	})
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.FetchNotNeeded {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.FetchNotNeeded)
	}
}

func TestCallNamingASmallerAgeStillAcceptsMarkdownTheConfiguredAgeAccepts(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			markdownStoredAt(time.Now().Add(-scrapeTolerance/2), storedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(context.Background(), pageread.PageCall{
		URL:          pageURLUnderTest(t),
		ToleratedAge: time.Second,
	})
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.FetchNotNeeded {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.FetchNotNeeded)
	}
}

func TestCallNamingAVersionAsksForNoScrape(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			markdownStoredAt(time.Now().Add(-2*scrapeTolerance), storedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(context.Background(), pageread.PageCall{
		URL:     pageURLUnderTest(t),
		Version: storedVersion,
	})
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.FetchNotNeeded {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.FetchNotNeeded)
	}
	if under.requests.asked != 0 {
		t.Errorf("scrape requests = %d, want none", under.requests.asked)
	}
}

func TestPageTheFetchCouldNotReadIsAnsweredAsNotReadable(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{{}}},
		&outcomesAnnouncing{outcome: pageread.PageNotReadable},
	)

	page, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.PageNotReadable {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.PageNotReadable)
	}
	if page.URL != pageURLUnderTest(t) {
		t.Errorf("page url = %q, want the url of the call", page.URL)
	}
	if page.Excerpt.Markdown != "" {
		t.Errorf("markdown = %q, want none", page.Excerpt.Markdown)
	}
}

func TestWaitThatRunsOutIsAnsweredAsUnfinished(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{{}}},
		&outcomesAnnouncing{outcome: pageread.FetchUnfinished},
	)

	page, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.FetchUnfinished {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.FetchUnfinished)
	}
	if under.progress.outcomesNotHeard != 1 {
		t.Errorf("outcomes not heard = %d, want 1", under.progress.outcomesNotHeard)
	}
}

func TestListenerThatCannotOpenIsAnsweredAsUnfinished(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{{}}},
		&outcomesAnnouncing{listenError: errors.New("broker away")},
	)

	page, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.FetchOutcome != pageread.FetchUnfinished {
		t.Errorf("fetch outcome = %q, want %q", page.FetchOutcome, pageread.FetchUnfinished)
	}
	if under.requests.asked != 0 {
		t.Errorf("scrape requests = %d, want none without a listener", under.requests.asked)
	}
}

func TestCorpusThatCannotBeReachedFailsTheCall(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{failure: errCorpusAway},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	_, err := under.reader.PageAnswerFor(
		context.Background(),
		pageread.PageCall{URL: pageURLUnderTest(t)},
	)
	if !errors.Is(err, errCorpusAway) {
		t.Fatalf("error = %v, want it to carry %v", err, errCorpusAway)
	}
	if under.progress.recallFailures != 1 {
		t.Errorf("recall failures = %d, want 1", under.progress.recallFailures)
	}
}

func TestAnswerCarriesOnlyTheCharactersTheCallAsksFor(t *testing.T) {
	under := newReaderUnderTest(
		&corpusHolding{stored: []pageread.PageMarkdown{
			markdownStoredAt(time.Now(), storedVersion),
		}},
		&outcomesAnnouncing{outcome: pageread.PageFetched},
	)

	page, err := under.reader.PageAnswerFor(context.Background(), pageread.PageCall{
		URL:            pageURLUnderTest(t),
		CharacterLimit: 4,
	})
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if page.Excerpt.Markdown != storedMarkdown[:4] {
		t.Errorf("markdown = %q, want %q", page.Excerpt.Markdown, storedMarkdown[:4])
	}
	if !page.Excerpt.Truncated {
		t.Error("the answer does not say it carries less than the whole markdown")
	}
}
