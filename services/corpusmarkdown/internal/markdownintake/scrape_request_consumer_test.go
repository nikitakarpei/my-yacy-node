package markdownintake_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
)

type fakeFetch struct {
	outcome pagefetch.FetchOutcome
	err     error
	mu      sync.Mutex
	urls    []string
}

func (f *fakeFetch) Fetch(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	f.mu.Lock()
	f.urls = append(f.urls, pageURL.String())
	f.mu.Unlock()
	return f.outcome, f.err
}

type recordingCorpus struct {
	fail     bool
	mu       sync.Mutex
	markdown map[string][]byte
}

func (c *recordingCorpus) Put(
	_ context.Context,
	scrapeRequestURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if c.fail {
		return errors.New("store failed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.markdown == nil {
		c.markdown = map[string][]byte{}
	}
	c.markdown[scrapeRequestURL.String()] = markdown
	return nil
}

type recordingRedirections struct {
	fail           bool
	mu             sync.Mutex
	byRequestedURL map[string]string
}

func (r *recordingRedirections) Record(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
) error {
	if r.fail {
		return errors.New("redirection not recorded")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byRequestedURL == nil {
		r.byRequestedURL = map[string]string{}
	}
	r.byRequestedURL[requestedURL.String()] = markdownURL.String()
	return nil
}

type recordingProgress struct {
	mu                       sync.Mutex
	received                 int
	originFetchFailures      int
	originFetchDeferrals     int
	extractionFailures       []string
	corpusWriteFailures      int
	redirectionWriteFailures int
	stored                   map[string]string
	nothingToScrape          []string
	noMarkdownDerived        []string
}

func (p *recordingProgress) ScrapeRequestReceived(_ context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.received++
}

func (p *recordingProgress) OriginFetchFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.originFetchFailures++
}

func (p *recordingProgress) OriginFetchDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.originFetchDeferrals++
}

func (p *recordingProgress) NothingToScrape(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nothingToScrape = append(p.nothingToScrape, requestedURL.String())
}

func (p *recordingProgress) DocumentExtractionFailed(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.extractionFailures = append(p.extractionFailures, requestedURL.String())
}

func (p *recordingProgress) NoMarkdownDerived(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.noMarkdownDerived = append(p.noMarkdownDerived, requestedURL.String())
}

func (p *recordingProgress) MarkdownCorpusWriteFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.corpusWriteFailures++
}

func (p *recordingProgress) RedirectionRecordWriteFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.redirectionWriteFailures++
}

func (p *recordingProgress) MarkdownStored(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stored == nil {
		p.stored = map[string]string{}
	}
	p.stored[requestedURL.String()] = markdownURL.String()
}

const scrapeRequestURL = "https://example.com/"

const article = `<!DOCTYPE html><html lang="en"><head><title>Sample Article</title></head>` +
	`<body><article><h1>Sample Article</h1><p>` + longText + `</p><p>` + longText +
	`</p></article></body></html>`

const longText = "The quick brown fox jumps over the lazy dog while the industrious " +
	"beaver builds a sturdy dam across the wide and winding river near the old mill town."

func scrapeRequestMessage(t *testing.T) *pullintaketest.Message {
	t.Helper()
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			PageURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &pullintaketest.Message{Body: data}
}

func archivedScrapeRequestMessage(
	t *testing.T,
	fetchURL string,
) *pullintaketest.Message {
	t.Helper()
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			PageURL:  canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
			FetchURL: canonicalurltest.CanonicalURLOf(t, fetchURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &pullintaketest.Message{Body: data}
}

func fetchOf(page pagefetch.FetchedPage) *fakeFetch {
	return &fakeFetch{
		outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded, Page: page},
	}
}

func fetchedPage(t *testing.T, contentType string, body string) pagefetch.FetchedPage {
	t.Helper()
	return fetchedPageAt(t, scrapeRequestURL, contentType, body)
}

func fetchedPageAt(
	t *testing.T,
	landedURL string,
	contentType string,
	body string,
) pagefetch.FetchedPage {
	t.Helper()
	return pagefetch.FetchedPage{
		LandedURL:   canonicalurltest.CanonicalURLOf(t, landedURL),
		ContentType: contentType,
		Body:        []byte(body),
	}
}

func run(
	t *testing.T,
	source pullintaketest.MessageSource,
	fetcher markdownintake.PageFetcher,
	corpus markdownintake.PageMarkdownCorpus,
	progress markdownintake.ScrapeProgress,
) error {
	t.Helper()
	return runRecording(t, intakeUnderTest{
		source:   source,
		fetcher:  fetcher,
		corpus:   corpus,
		progress: progress,
	})
}

type intakeUnderTest struct {
	source       pullintaketest.MessageSource
	fetcher      markdownintake.PageFetcher
	corpus       markdownintake.PageMarkdownCorpus
	progress     markdownintake.ScrapeProgress
	redirections markdownintake.PageRedirections
}

func runRecording(t *testing.T, intake intakeUnderTest) error {
	t.Helper()
	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page formats: %v", err)
	}
	if intake.redirections == nil {
		intake.redirections = &recordingRedirections{}
	}
	if intake.progress == nil {
		intake.progress = &recordingProgress{}
	}
	if intake.corpus == nil {
		intake.corpus = &recordingCorpus{}
	}
	return markdownintake.NewScrapeRequestConsumer(markdownintake.Config{
		Source:                         intake.source,
		Fetcher:                        intake.fetcher,
		FormatDerivations:              formatDerivations,
		Corpus:                         intake.corpus,
		Redirections:                   intake.redirections,
		Progress:                       intake.progress,
		ScrapeRequestIntakeConcurrency: 1,
	}).Run(context.Background())
}

const redirectedToURL = "https://www.example.com/"

func TestConsumerStoresARedirectedPageUnderTheURLTheOriginSettledOn(t *testing.T) {
	message := scrapeRequestMessage(t)
	corpus := &recordingCorpus{}
	redirections := &recordingRedirections{}

	if err := runRecording(t, intakeUnderTest{
		source:       pullintaketest.MessageSourceOf(message),
		fetcher:      fetchOf(fetchedPageAt(t, redirectedToURL, "text/html", article)),
		corpus:       corpus,
		redirections: redirections,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if _, stored := corpus.markdown[redirectedToURL]; !stored {
		t.Errorf("stored under %v, want the url the origin settled on", corpus.markdown)
	}
	if redirections.byRequestedURL[scrapeRequestURL] != redirectedToURL {
		t.Errorf("redirections = %v, want the requested url leading to the settled one",
			redirections.byRequestedURL)
	}
}

func TestConsumerReadsAPageFromTheFetchURLAndStoresItUnderThePageURL(t *testing.T) {
	const replayURL = "http://archive.example/replay/https://example.com/"
	message := archivedScrapeRequestMessage(t, replayURL)
	corpus := &recordingCorpus{}
	redirections := &recordingRedirections{}
	fetcher := fetchOf(fetchedPageAt(t, replayURL, "text/html", article))

	if err := runRecording(t, intakeUnderTest{
		source:       pullintaketest.MessageSourceOf(message),
		fetcher:      fetcher,
		corpus:       corpus,
		redirections: redirections,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != replayURL {
		t.Errorf("fetched %v, want the fetch url", fetcher.urls)
	}
	if _, stored := corpus.markdown[scrapeRequestURL]; !stored {
		t.Errorf("stored under %v, want the page url", corpus.markdown)
	}
	if len(redirections.byRequestedURL) != 0 {
		t.Errorf("redirections = %v, want none", redirections.byRequestedURL)
	}
}

func TestConsumerRecordsNoRedirectionForAPageThatDidNotRedirect(t *testing.T) {
	message := scrapeRequestMessage(t)
	redirections := &recordingRedirections{}

	if err := runRecording(t, intakeUnderTest{
		source:       pullintaketest.MessageSourceOf(message),
		fetcher:      fetchOf(fetchedPage(t, "text/html", article)),
		redirections: redirections,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(redirections.byRequestedURL) != 0 {
		t.Errorf("redirections = %v, want none", redirections.byRequestedURL)
	}
}

func TestConsumerReturnsAScrapeRequestWhoseRedirectionCannotBeRecorded(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := runRecording(t, intakeUnderTest{
		source:       pullintaketest.MessageSourceOf(message),
		fetcher:      fetchOf(fetchedPageAt(t, redirectedToURL, "text/html", article)),
		progress:     progress,
		redirections: &recordingRedirections{fail: true},
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if len(progress.stored) != 0 || progress.redirectionWriteFailures != 1 {
		t.Errorf("progress = %+v, want nothing stored and one redirection write failure", progress)
	}
}

func TestConsumerReportsTheStoredMarkdownUnderTheRequestedURL(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := runRecording(t, intakeUnderTest{
		source:   pullintaketest.MessageSourceOf(message),
		fetcher:  fetchOf(fetchedPageAt(t, redirectedToURL, "text/html", article)),
		progress: progress,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if progress.stored[scrapeRequestURL] != redirectedToURL {
		t.Errorf("reported %v, want the requested url carrying the settled one", progress.stored)
	}
	if len(progress.extractionFailures) != 0 || len(progress.nothingToScrape) != 0 {
		t.Errorf("progress = %+v, want no give-up reported", progress)
	}
}

func TestConsumerReportsDocumentExtractionFailedForAPageItCannotExtract(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := runRecording(t, intakeUnderTest{
		source:   pullintaketest.MessageSourceOf(message),
		fetcher:  fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")),
		progress: progress,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(progress.extractionFailures) != 1 || progress.extractionFailures[0] != scrapeRequestURL {
		t.Errorf("reported document extraction failed for %v, want the requested url",
			progress.extractionFailures)
	}
}

func TestConsumerStoresTheMarkdownItDerivesFromAScrapeRequest(t *testing.T) {
	message := scrapeRequestMessage(t)
	fetcher := fetchOf(fetchedPage(t, "text/html", article))
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(fetcher.urls) != 1 || fetcher.urls[0] != scrapeRequestURL {
		t.Errorf("fetched %v, want the scrape request url", fetcher.urls)
	}
	if got := string(corpus.markdown[scrapeRequestURL]); !strings.Contains(got, "quick brown fox") {
		t.Errorf("stored = %q, want the article as markdown", got)
	}
	if progress.received != 1 || len(progress.stored) != 1 {
		t.Errorf("progress = %+v, want one received and one stored", progress)
	}
}

func TestConsumerAcksAScrapeRequestHoldingNoExtractableDocument(t *testing.T) {
	message := scrapeRequestMessage(t)
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")), corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(corpus.markdown) != 0 || len(progress.stored) != 0 {
		t.Errorf("stored %v, want nothing", corpus.markdown)
	}
}

func TestConsumerNaksWhenTheFetchBreaks(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		&fakeFetch{err: errors.New("fetch broke")}, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.originFetchFailures != 1 || len(progress.stored) != 0 {
		t.Errorf("progress = %+v, want one origin fetch failure", progress)
	}
}

func TestConsumerNaksWhenTheFetchReportsFailure(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.originFetchFailures != 1 {
		t.Errorf("progress = %+v, want one origin fetch failure", progress)
	}
}

func TestConsumerHoldsADeferredScrapeRequestBackForAsLongAsTheOriginAsks(t *testing.T) {
	message := scrapeRequestMessage(t)
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: 30 * time.Second,
	}}

	progress := &recordingProgress{}

	if err := run(
		t,
		pullintaketest.MessageSourceOf(message),
		fetcher,
		&recordingCorpus{},
		progress,
	); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak-with-delay", action)
	}
	if message.HeldBackFor() != 30*time.Second {
		t.Errorf("nak delay = %v, want the deferral the origin asked for", message.HeldBackFor())
	}
	if progress.originFetchDeferrals != 1 || progress.originFetchFailures != 0 {
		t.Errorf("progress = %+v, want one deferral and no failure", progress)
	}
}

func TestConsumerAcksAScrapeRequestThatFetchesNoPage(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(progress.nothingToScrape) != 1 || progress.nothingToScrape[0] != scrapeRequestURL {
		t.Errorf("reported nothing to scrape for %v, want the requested url",
			progress.nothingToScrape)
	}
	if len(progress.stored) != 0 || progress.originFetchFailures != 0 {
		t.Errorf("progress = %+v, want nothing stored and no failure", progress)
	}
}

func TestConsumerNaksWhenTheStoreFails(t *testing.T) {
	message := scrapeRequestMessage(t)
	progress := &recordingProgress{}

	if err := run(t, pullintaketest.MessageSourceOf(message),
		fetchOf(fetchedPage(t, "text/html", article)),
		&recordingCorpus{fail: true}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.corpusWriteFailures != 1 || len(progress.stored) != 0 {
		t.Errorf("progress = %+v, want one store failure", progress)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not json")}
	progress := &recordingProgress{}

	err := run(t, pullintaketest.MessageSourceOf(message),
		&fakeFetch{}, &recordingCorpus{}, progress)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("run error = %v, want poison halt", err)
	}
	if settled := message.Settlements(); len(settled) != 0 {
		t.Fatalf("undecodable message settled %v, want left pending", settled)
	}
	if progress.received != 1 {
		t.Errorf("progress = %+v, want one received", progress)
	}
}
