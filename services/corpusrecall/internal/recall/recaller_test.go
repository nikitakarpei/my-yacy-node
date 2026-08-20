package recall_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	kindMarkdown     recall.RepresentationKind = "markdown"
	kindText         recall.RepresentationKind = "text"
	recalledMarkdown                           = "# recalled page"
	requestedURL                               = "https://example.com"
)

type markdownPage struct {
	markdown string
}

type recordingCrawlOrders struct {
	orderedURLs []string
	err         error
}

func (c *recordingCrawlOrders) Place(
	_ context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
) error {
	c.orderedURLs = append(c.orderedURLs, canonicalURL.String())
	return c.err
}

type blockingCrawlOrders struct {
	entered chan struct{}
	release chan struct{}
}

func (c *blockingCrawlOrders) Place(_ context.Context, _ yacycrawlcontract.CanonicalURL) error {
	close(c.entered)
	<-c.release
	return nil
}

type unredirectedURL struct{}

func (unredirectedURL) ResolvedURLOf(
	_ context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
) (yacycrawlcontract.CanonicalURL, error) {
	return canonicalURL, nil
}

type failingRedirects struct{}

func (failingRedirects) ResolvedURLOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (yacycrawlcontract.CanonicalURL, error) {
	return yacycrawlcontract.CanonicalURL{}, errors.New("redirect bucket down")
}

type emptyCorpus struct{}

func (emptyCorpus) RepresentationKind() recall.RepresentationKind { return kindMarkdown }

func (emptyCorpus) RepresentationOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (recall.Representation, bool, error) {
	return nil, false, nil
}

type filledCorpus struct {
	mu              sync.Mutex
	reads           int
	readsBeforeFill int
}

func (*filledCorpus) RepresentationKind() recall.RepresentationKind { return kindMarkdown }

func (c *filledCorpus) RepresentationOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (recall.Representation, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	if c.reads > c.readsBeforeFill {
		return markdownPage{markdown: recalledMarkdown}, true, nil
	}
	return nil, false, nil
}

type failingCorpus struct{}

func (failingCorpus) RepresentationKind() recall.RepresentationKind { return kindMarkdown }

func (failingCorpus) RepresentationOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (recall.Representation, bool, error) {
	return nil, false, errors.New("corpus down")
}

type keptPages struct{}

func (keptPages) DisposalMarkOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (recall.DisposalMark, error) {
	return "", nil
}

func (keptPages) DisposalOccurredSince(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
	_ recall.DisposalMark,
) (bool, error) {
	return false, nil
}

type pageDisposedDuringRecall struct{}

func (pageDisposedDuringRecall) DisposalMarkOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (recall.DisposalMark, error) {
	return "", nil
}

func (pageDisposedDuringRecall) DisposalOccurredSince(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
	_ recall.DisposalMark,
) (bool, error) {
	return true, nil
}

type failingDisposalLookup struct {
	failsAtRecallStart bool
}

func (d failingDisposalLookup) DisposalMarkOf(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
) (recall.DisposalMark, error) {
	if d.failsAtRecallStart {
		return "", errors.New("disposal bucket down")
	}
	return "", nil
}

func (failingDisposalLookup) DisposalOccurredSince(
	_ context.Context,
	_ yacycrawlcontract.CanonicalURL,
	_ recall.DisposalMark,
) (bool, error) {
	return false, errors.New("disposal bucket down")
}

type recordingProgress struct {
	requestsAccepted int
	requestsRejected int
	recalledKinds    []recall.RepresentationKind
	unavailableKinds []recall.RepresentationKind
}

func (p *recordingProgress) RequestAccepted() { p.requestsAccepted++ }
func (p *recordingProgress) RequestRejected() { p.requestsRejected++ }
func (p *recordingProgress) RepresentationRecalled(kind recall.RepresentationKind) {
	p.recalledKinds = append(p.recalledKinds, kind)
}

func (p *recordingProgress) RepresentationUnavailable(kind recall.RepresentationKind) {
	p.unavailableKinds = append(p.unavailableKinds, kind)
}

type recallerBuilder struct {
	crawlOrders         recall.CrawlOrderPlacer
	redirects           recall.RedirectResolutions
	disposedPages       recall.DisposedPages
	markdownCorpus      recall.Corpus
	progress            *recordingProgress
	recallLimit         time.Duration
	maxRequestsInFlight int
}

func (r recallerBuilder) recaller(t *testing.T) *recall.Recaller {
	t.Helper()
	if r.crawlOrders == nil {
		r.crawlOrders = &recordingCrawlOrders{}
	}
	if r.redirects == nil {
		r.redirects = unredirectedURL{}
	}
	if r.disposedPages == nil {
		r.disposedPages = keptPages{}
	}
	if r.progress == nil {
		r.progress = &recordingProgress{}
	}
	if r.recallLimit == 0 {
		r.recallLimit = time.Second
	}
	if r.maxRequestsInFlight == 0 {
		r.maxRequestsInFlight = 4
	}
	var corpora []recall.Corpus
	if r.markdownCorpus != nil {
		corpora = append(corpora, r.markdownCorpus)
	}
	recaller, err := recall.NewRecaller(
		r.crawlOrders,
		r.redirects,
		r.disposedPages,
		corpora,
		r.progress,
		recall.Config{
			RecallLimit:         r.recallLimit,
			PollInterval:        time.Millisecond,
			MaxRequestsInFlight: r.maxRequestsInFlight,
		},
	)
	if err != nil {
		t.Fatalf("new recaller: %v", err)
	}
	return recaller
}

func recalledMarkdownPage(t *testing.T, recaller *recall.Recaller) recall.RecalledPage {
	t.Helper()
	recalled, err := recaller.Recall(
		context.Background(), requestedURL, []recall.RepresentationKind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	return recalled
}

func assertOnlyUnavailableKind(
	t *testing.T,
	recalled recall.RecalledPage,
	kind recall.RepresentationKind,
) {
	t.Helper()
	if len(recalled.Representations) != 0 {
		t.Fatalf("recalled representations = %+v, want none", recalled.Representations)
	}
	if len(recalled.UnavailableKinds) != 1 || recalled.UnavailableKinds[0] != kind {
		t.Fatalf("unavailable kinds = %v, want [%s]", recalled.UnavailableKinds, kind)
	}
}

func assertOnlyRecalledMarkdown(t *testing.T, recalled recall.RecalledPage) {
	t.Helper()
	if len(recalled.UnavailableKinds) != 0 {
		t.Fatalf("unavailable kinds = %v, want none", recalled.UnavailableKinds)
	}
	if len(recalled.Representations) != 1 {
		t.Fatalf("recalled representations = %+v, want one", recalled.Representations)
	}
	recalledMarkdownRepresentation := recall.RecalledRepresentation{
		Kind:           kindMarkdown,
		Representation: markdownPage{markdown: recalledMarkdown},
	}
	if recalled.Representations[0] != recalledMarkdownRepresentation {
		t.Fatalf("recalled representation = %+v", recalled.Representations[0])
	}
}

func TestRecallerIsRefusedWhenTwoCorporaServeOneRepresentationKind(t *testing.T) {
	_, err := recall.NewRecaller(
		&recordingCrawlOrders{},
		unredirectedURL{},
		keptPages{},
		[]recall.Corpus{emptyCorpus{}, &filledCorpus{}},
		&recordingProgress{},
		recall.Config{},
	)

	if !errors.Is(err, recall.ErrRepresentationKindServedTwice) {
		t.Fatalf("new recaller error = %v", err)
	}
}

func TestRecallReturnsTheRepresentationTheCorpusHolds(t *testing.T) {
	crawlOrders := &recordingCrawlOrders{}
	progress := &recordingProgress{}
	recaller := recallerBuilder{
		crawlOrders:    crawlOrders,
		markdownCorpus: &filledCorpus{},
		progress:       progress,
	}.recaller(t)

	assertOnlyRecalledMarkdown(t, recalledMarkdownPage(t, recaller))

	if len(crawlOrders.orderedURLs) != 1 || crawlOrders.orderedURLs[0] != requestedURL+"/" {
		t.Errorf("ordered urls = %v", crawlOrders.orderedURLs)
	}
	if progress.requestsAccepted != 1 {
		t.Errorf("requests accepted = %d, want 1", progress.requestsAccepted)
	}
	if len(progress.recalledKinds) != 1 || progress.recalledKinds[0] != kindMarkdown {
		t.Errorf("recalled kinds = %v", progress.recalledKinds)
	}
}

func TestRecallReturnsTheRepresentationTheCorpusGainsWhileWaiting(t *testing.T) {
	recaller := recallerBuilder{markdownCorpus: &filledCorpus{readsBeforeFill: 2}}.recaller(t)

	assertOnlyRecalledMarkdown(t, recalledMarkdownPage(t, recaller))
}

func TestRecallReportsAKindNoCorpusServesUnavailable(t *testing.T) {
	progress := &recordingProgress{}
	recaller := recallerBuilder{markdownCorpus: &filledCorpus{}, progress: progress}.recaller(t)

	recalled, err := recaller.Recall(
		context.Background(), requestedURL, []recall.RepresentationKind{kindText},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}

	assertOnlyUnavailableKind(t, recalled, kindText)
	if len(progress.unavailableKinds) != 1 || progress.unavailableKinds[0] != kindText {
		t.Errorf("unavailable kinds = %v", progress.unavailableKinds)
	}
}

func TestRecallReportsAKindTheCorpusNeverHoldsUnavailableAtTheRecallLimit(t *testing.T) {
	recaller := recallerBuilder{
		markdownCorpus: emptyCorpus{},
		recallLimit:    20 * time.Millisecond,
	}.recaller(t)

	assertOnlyUnavailableKind(t, recalledMarkdownPage(t, recaller), kindMarkdown)
}

func TestRecallReportsAKindUnavailableWhenTheCorpusReadFails(t *testing.T) {
	recaller := recallerBuilder{markdownCorpus: failingCorpus{}}.recaller(t)

	assertOnlyUnavailableKind(t, recalledMarkdownPage(t, recaller), kindMarkdown)
}

func TestRecallReportsAKindUnavailableWhenTheRedirectLookupFails(t *testing.T) {
	recaller := recallerBuilder{
		redirects:      failingRedirects{},
		markdownCorpus: &filledCorpus{},
	}.recaller(t)

	assertOnlyUnavailableKind(t, recalledMarkdownPage(t, recaller), kindMarkdown)
}

func TestRecallStopsWaitingWhenThePageIsDisposedOfDuringTheRecall(t *testing.T) {
	recaller := recallerBuilder{
		disposedPages:  pageDisposedDuringRecall{},
		markdownCorpus: emptyCorpus{},
		recallLimit:    time.Minute,
	}.recaller(t)

	start := time.Now()
	assertOnlyUnavailableKind(t, recalledMarkdownPage(t, recaller), kindMarkdown)

	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("recall waited %v, want a stop ahead of the recall limit", elapsed)
	}
}

func TestRecallKeepsWaitingWhenTheDisposalLookupFailsMidRecall(t *testing.T) {
	recaller := recallerBuilder{
		disposedPages:  failingDisposalLookup{},
		markdownCorpus: &filledCorpus{readsBeforeFill: 2},
	}.recaller(t)

	assertOnlyRecalledMarkdown(t, recalledMarkdownPage(t, recaller))
}

func TestRecallFailsWhenTheDisposalLookupFailsAtTheStart(t *testing.T) {
	recaller := recallerBuilder{
		disposedPages:  failingDisposalLookup{failsAtRecallStart: true},
		markdownCorpus: &filledCorpus{},
	}.recaller(t)

	if _, err := recaller.Recall(
		context.Background(), requestedURL, []recall.RepresentationKind{kindMarkdown},
	); err == nil {
		t.Fatal("expected a disposal lookup error")
	}
}

func TestRecallFailsWhenTheURLCannotBeCanonicalized(t *testing.T) {
	recaller := recallerBuilder{markdownCorpus: &filledCorpus{}}.recaller(t)

	if _, err := recaller.Recall(context.Background(), "://nonsense", nil); err == nil {
		t.Fatal("expected a canonicalization error")
	}
}

func TestRecallFailsWhenTheCrawlOrderCannotBePlaced(t *testing.T) {
	recaller := recallerBuilder{
		crawlOrders:    &recordingCrawlOrders{err: errors.New("no stream")},
		markdownCorpus: &filledCorpus{},
	}.recaller(t)

	if _, err := recaller.Recall(
		context.Background(), requestedURL, []recall.RepresentationKind{kindMarkdown},
	); err == nil {
		t.Fatal("expected a crawl order placement error")
	}
}

func TestRecallRejectsARequestBeyondTheInFlightLimit(t *testing.T) {
	blocking := &blockingCrawlOrders{entered: make(chan struct{}), release: make(chan struct{})}
	progress := &recordingProgress{}
	recaller := recallerBuilder{
		crawlOrders:         blocking,
		markdownCorpus:      &filledCorpus{},
		progress:            progress,
		maxRequestsInFlight: 1,
	}.recaller(t)

	go func() {
		_, _ = recaller.Recall(
			context.Background(), requestedURL, []recall.RepresentationKind{kindMarkdown},
		)
	}()
	<-blocking.entered

	_, err := recaller.Recall(
		context.Background(), requestedURL, []recall.RepresentationKind{kindMarkdown},
	)
	if !errors.Is(err, recall.ErrTooManyRequestsInFlight) {
		t.Fatalf("err = %v, want ErrTooManyRequestsInFlight", err)
	}
	if progress.requestsRejected != 1 {
		t.Errorf("requests rejected = %d, want 1", progress.requestsRejected)
	}
	close(blocking.release)
}
