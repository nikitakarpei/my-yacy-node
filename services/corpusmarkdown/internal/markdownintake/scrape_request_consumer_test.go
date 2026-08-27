package markdownintake_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
)

type fakeMsg struct {
	data     []byte
	acked    chan string
	nakDelay time.Duration
}

func (m *fakeMsg) Subject() string                 { return "subj" }
func (m *fakeMsg) Reply() string                   { return "" }
func (m *fakeMsg) Data() []byte                    { return m.data }
func (m *fakeMsg) Headers() nats.Header            { return nil }
func (m *fakeMsg) Ack() error                      { m.acked <- "ack"; return nil }
func (m *fakeMsg) DoubleAck(context.Context) error { m.acked <- "ack"; return nil }
func (m *fakeMsg) Nak() error                      { m.acked <- "nak"; return nil }
func (m *fakeMsg) InProgress() error               { return nil }
func (m *fakeMsg) Term() error                     { m.acked <- "term"; return nil }
func (m *fakeMsg) TermWithReason(string) error     { m.acked <- "term"; return nil }

func (m *fakeMsg) NakWithDelay(delay time.Duration) error {
	m.nakDelay = delay
	m.acked <- "nak-with-delay"
	return nil
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{}, nil
}

type fakeIterator struct {
	messages []jetstream.Msg
	mu       sync.Mutex
}

func (it *fakeIterator) Next(...jetstream.NextOpt) (jetstream.Msg, error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if len(it.messages) == 0 {
		return nil, jetstream.ErrMsgIteratorClosed
	}
	msg := it.messages[0]
	it.messages = it.messages[1:]
	return msg, nil
}

func (it *fakeIterator) Stop()  {}
func (it *fakeIterator) Drain() {}

type fakeSource struct {
	iterator *fakeIterator
}

func (s fakeSource) Messages(...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error) {
	return s.iterator, nil
}

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

func scrapeRequestMessage(t *testing.T, acked chan string) *fakeMsg {
	t.Helper()
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			PageURL: canonicalurltest.CanonicalURLOf(t, scrapeRequestURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &fakeMsg{data: data, acked: acked}
}

func archivedScrapeRequestMessage(t *testing.T, acked chan string, fetchURL string) *fakeMsg {
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
	return &fakeMsg{data: data, acked: acked}
}

func sourceOf(messages ...jetstream.Msg) fakeSource {
	return fakeSource{iterator: &fakeIterator{messages: messages}}
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
	source fakeSource,
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
	source       fakeSource
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
	acked := make(chan string, 1)
	corpus := &recordingCorpus{}
	redirections := &recordingRedirections{}

	if err := runRecording(t, intakeUnderTest{
		source:       sourceOf(scrapeRequestMessage(t, acked)),
		fetcher:      fetchOf(fetchedPageAt(t, redirectedToURL, "text/html", article)),
		corpus:       corpus,
		redirections: redirections,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
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
	acked := make(chan string, 1)
	corpus := &recordingCorpus{}
	redirections := &recordingRedirections{}
	fetcher := fetchOf(fetchedPageAt(t, replayURL, "text/html", article))

	if err := runRecording(t, intakeUnderTest{
		source:       sourceOf(archivedScrapeRequestMessage(t, acked, replayURL)),
		fetcher:      fetcher,
		corpus:       corpus,
		redirections: redirections,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
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
	acked := make(chan string, 1)
	redirections := &recordingRedirections{}

	if err := runRecording(t, intakeUnderTest{
		source:       sourceOf(scrapeRequestMessage(t, acked)),
		fetcher:      fetchOf(fetchedPage(t, "text/html", article)),
		redirections: redirections,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	<-acked
	if len(redirections.byRequestedURL) != 0 {
		t.Errorf("redirections = %v, want none", redirections.byRequestedURL)
	}
}

func TestConsumerReturnsAScrapeRequestWhoseRedirectionCannotBeRecorded(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := runRecording(t, intakeUnderTest{
		source:       sourceOf(scrapeRequestMessage(t, acked)),
		fetcher:      fetchOf(fetchedPageAt(t, redirectedToURL, "text/html", article)),
		progress:     progress,
		redirections: &recordingRedirections{fail: true},
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if len(progress.stored) != 0 || progress.redirectionWriteFailures != 1 {
		t.Errorf("progress = %+v, want nothing stored and one redirection write failure", progress)
	}
}

func TestConsumerReportsTheStoredMarkdownUnderTheRequestedURL(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := runRecording(t, intakeUnderTest{
		source:   sourceOf(scrapeRequestMessage(t, acked)),
		fetcher:  fetchOf(fetchedPageAt(t, redirectedToURL, "text/html", article)),
		progress: progress,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	<-acked
	if progress.stored[scrapeRequestURL] != redirectedToURL {
		t.Errorf("reported %v, want the requested url carrying the settled one", progress.stored)
	}
	if len(progress.extractionFailures) != 0 || len(progress.nothingToScrape) != 0 {
		t.Errorf("progress = %+v, want no give-up reported", progress)
	}
}

func TestConsumerReportsDocumentExtractionFailedForAPageItCannotExtract(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := runRecording(t, intakeUnderTest{
		source:   sourceOf(scrapeRequestMessage(t, acked)),
		fetcher:  fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")),
		progress: progress,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	<-acked
	if len(progress.extractionFailures) != 1 || progress.extractionFailures[0] != scrapeRequestURL {
		t.Errorf("reported document extraction failed for %v, want the requested url",
			progress.extractionFailures)
	}
}

func TestConsumerStoresTheMarkdownItDerivesFromAScrapeRequest(t *testing.T) {
	acked := make(chan string, 1)
	fetcher := fetchOf(fetchedPage(t, "text/html", article))
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
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
	acked := make(chan string, 1)
	corpus := &recordingCorpus{}
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetchOf(fetchedPage(t, "application/pdf", "%PDF-1.4")), corpus, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
		t.Errorf("action = %q, want ack", action)
	}
	if len(corpus.markdown) != 0 || len(progress.stored) != 0 {
		t.Errorf("stored %v, want nothing", corpus.markdown)
	}
}

func TestConsumerNaksWhenTheFetchBreaks(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		&fakeFetch{err: errors.New("fetch broke")}, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.originFetchFailures != 1 || len(progress.stored) != 0 {
		t.Errorf("progress = %+v, want one origin fetch failure", progress)
	}
}

func TestConsumerNaksWhenTheFetchReportsFailure(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.originFetchFailures != 1 {
		t.Errorf("progress = %+v, want one origin fetch failure", progress)
	}
}

func TestConsumerHoldsADeferredScrapeRequestBackForAsLongAsTheOriginAsks(t *testing.T) {
	acked := make(chan string, 1)
	message := scrapeRequestMessage(t, acked)
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{
		Status:   pagefetch.FetchDeferred,
		DeferFor: 30 * time.Second,
	}}

	progress := &recordingProgress{}

	if err := run(t, sourceOf(message), fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak-with-delay", action)
	}
	if message.nakDelay != 30*time.Second {
		t.Errorf("nak delay = %v, want the deferral the origin asked for", message.nakDelay)
	}
	if progress.originFetchDeferrals != 1 || progress.originFetchFailures != 0 {
		t.Errorf("progress = %+v, want one deferral and no failure", progress)
	}
}

func TestConsumerAcksAScrapeRequestThatFetchesNoPage(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}
	fetcher := &fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetcher, &recordingCorpus{}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "ack" {
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
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	if err := run(t, sourceOf(scrapeRequestMessage(t, acked)),
		fetchOf(fetchedPage(t, "text/html", article)),
		&recordingCorpus{fail: true}, progress); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := <-acked; action != "nak-with-delay" {
		t.Errorf("action = %q, want nak", action)
	}
	if progress.corpusWriteFailures != 1 || len(progress.stored) != 0 {
		t.Errorf("progress = %+v, want one store failure", progress)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	acked := make(chan string, 1)
	progress := &recordingProgress{}

	err := run(t, sourceOf(&fakeMsg{data: []byte("not json"), acked: acked}),
		&fakeFetch{}, &recordingCorpus{}, progress)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("run error = %v, want poison halt", err)
	}
	select {
	case action := <-acked:
		t.Fatalf("undecodable message was %q, want left pending", action)
	default:
	}
	if progress.received != 1 {
		t.Errorf("progress = %+v, want one received", progress)
	}
}
