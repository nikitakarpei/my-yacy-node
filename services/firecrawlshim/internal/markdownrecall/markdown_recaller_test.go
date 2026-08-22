package markdownrecall_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/markdownrecall"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
)

const (
	requestedURL      = "https://example.com"
	canonicalURL      = "https://example.com/"
	resolvedURL       = "https://example.com/moved"
	recalledMarkdown  = "# recalled page"
	disposalReason    = "not-a-page"
	disposalMarkFirst = "00000000000000000001"
	disposalMarkNext  = "00000000000000000002"
)

type crawlOutcomes struct {
	mu                 sync.Mutex
	reads              int
	requestedURLs      []string
	disposal           *crawlerv1.PageDisposal
	disposalAfterReads int
	disposedReason     string
	failWith           error
}

func (o *crawlOutcomes) ReadPage(
	_ context.Context,
	in *crawlerv1.ReadPageRequest,
	_ ...grpc.CallOption,
) (*crawlerv1.PageOutcome, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.failWith != nil {
		return nil, o.failWith
	}
	o.reads++
	o.requestedURLs = append(o.requestedURLs, in.GetUrl())
	outcome := &crawlerv1.PageOutcome{
		CanonicalUrl: canonicalURL,
		ResolvedUrl:  resolvedURL,
		Disposal:     o.disposal,
	}
	if o.disposedReason != "" && o.reads > o.disposalAfterReads {
		outcome.Disposal = &crawlerv1.PageDisposal{
			Mark:   disposalMarkNext,
			Reason: o.disposedReason,
		}
	}
	return outcome, nil
}

type markdownCorpus struct {
	mu              sync.Mutex
	reads           int
	requestedURLs   []string
	readsBeforeFill int
	failWith        error
}

func (c *markdownCorpus) RecallPage(
	_ context.Context,
	in *corpusmarkdownv1.RecallPageRequest,
	_ ...grpc.CallOption,
) (*corpusmarkdownv1.RecallPageResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failWith != nil {
		return nil, c.failWith
	}
	c.reads++
	c.requestedURLs = append(c.requestedURLs, in.GetUrl())
	if c.reads <= c.readsBeforeFill {
		return nil, status.Error(codes.NotFound, "no markdown")
	}
	return &corpusmarkdownv1.RecallPageResponse{
		CanonicalUrl: canonicalURL,
		Markdown:     recalledMarkdown,
	}, nil
}

type emptyMarkdownCorpus struct{}

func (emptyMarkdownCorpus) RecallPage(
	_ context.Context,
	_ *corpusmarkdownv1.RecallPageRequest,
	_ ...grpc.CallOption,
) (*corpusmarkdownv1.RecallPageResponse, error) {
	return nil, status.Error(codes.NotFound, "no markdown")
}

type crawlOrders struct {
	mu          sync.Mutex
	orderedURLs []string
	failWith    error
}

func (o *crawlOrders) Place(_ context.Context, canonicalURL string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.orderedURLs = append(o.orderedURLs, canonicalURL)
	return o.failWith
}

type blockingCrawlOrders struct {
	entered chan struct{}
	release chan struct{}
}

func (o *blockingCrawlOrders) Place(_ context.Context, _ string) error {
	close(o.entered)
	<-o.release
	return nil
}

type recallerBuilder struct {
	crawlOutcomes  markdownrecall.CrawlOutcomes
	crawlOrders    markdownrecall.CrawlOrders
	markdownCorpus markdownrecall.MarkdownCorpus
	recallLimit    time.Duration
	maxInFlight    int
}

func (b recallerBuilder) recaller() *markdownrecall.MarkdownRecaller {
	if b.crawlOutcomes == nil {
		b.crawlOutcomes = &crawlOutcomes{}
	}
	if b.crawlOrders == nil {
		b.crawlOrders = &crawlOrders{}
	}
	if b.markdownCorpus == nil {
		b.markdownCorpus = &markdownCorpus{}
	}
	if b.recallLimit == 0 {
		b.recallLimit = time.Second
	}
	if b.maxInFlight == 0 {
		b.maxInFlight = 4
	}
	return markdownrecall.NewMarkdownRecaller(
		b.crawlOutcomes,
		b.crawlOrders,
		b.markdownCorpus,
		markdownrecall.Config{
			RecallLimit:        b.recallLimit,
			PollInterval:       time.Millisecond,
			MaxRecallsInFlight: b.maxInFlight,
		},
	)
}

func TestRecallPageYieldsTheMarkdownTheCorpusHolds(t *testing.T) {
	orders := &crawlOrders{}
	corpus := &markdownCorpus{}
	recaller := recallerBuilder{crawlOrders: orders, markdownCorpus: corpus}.recaller()

	recalled, err := recaller.RecallPage(context.Background(), requestedURL)
	if err != nil {
		t.Fatalf("recall page: %v", err)
	}

	if recalled.Markdown != recalledMarkdown || recalled.CanonicalURL != canonicalURL {
		t.Errorf("recalled = %+v", recalled)
	}
	if len(orders.orderedURLs) != 1 || orders.orderedURLs[0] != canonicalURL {
		t.Errorf("ordered urls = %v", orders.orderedURLs)
	}
	if len(corpus.requestedURLs) != 1 || corpus.requestedURLs[0] != resolvedURL {
		t.Errorf("corpus was asked for %v, want the resolved url", corpus.requestedURLs)
	}
}

func TestRecallPageYieldsTheMarkdownTheCorpusGainsWhileWaiting(t *testing.T) {
	recaller := recallerBuilder{markdownCorpus: &markdownCorpus{readsBeforeFill: 2}}.recaller()

	recalled, err := recaller.RecallPage(context.Background(), requestedURL)
	if err != nil {
		t.Fatalf("recall page: %v", err)
	}
	if recalled.Markdown != recalledMarkdown {
		t.Errorf("markdown = %q", recalled.Markdown)
	}
}

func TestRecallPageReportsTheMarkdownUnavailableAtTheRecallLimit(t *testing.T) {
	recaller := recallerBuilder{
		markdownCorpus: emptyMarkdownCorpus{},
		recallLimit:    20 * time.Millisecond,
	}.recaller()

	_, err := recaller.RecallPage(context.Background(), requestedURL)

	if !errors.Is(err, markdownrecall.ErrMarkdownUnavailable) {
		t.Fatalf("err = %v, want ErrMarkdownUnavailable", err)
	}
}

func TestRecallPageStopsWaitingWhenCrawlingDisposesOfThePage(t *testing.T) {
	recaller := recallerBuilder{
		crawlOutcomes: &crawlOutcomes{
			disposal:           &crawlerv1.PageDisposal{Mark: disposalMarkFirst},
			disposalAfterReads: 1,
			disposedReason:     disposalReason,
		},
		markdownCorpus: emptyMarkdownCorpus{},
		recallLimit:    time.Minute,
	}.recaller()

	start := time.Now()
	_, err := recaller.RecallPage(context.Background(), requestedURL)

	if !errors.Is(err, markdownrecall.ErrMarkdownUnavailable) {
		t.Fatalf("err = %v, want ErrMarkdownUnavailable", err)
	}
	if !strings.Contains(err.Error(), disposalReason) {
		t.Errorf("err = %v, want the disposal reason in it", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("recall waited %v, want a stop ahead of the recall limit", elapsed)
	}
}

func TestRecallPageKeepsWaitingWhileAnEarlierDisposalStands(t *testing.T) {
	recaller := recallerBuilder{
		crawlOutcomes: &crawlOutcomes{
			disposal: &crawlerv1.PageDisposal{Mark: disposalMarkFirst, Reason: disposalReason},
		},
		markdownCorpus: &markdownCorpus{readsBeforeFill: 2},
	}.recaller()

	recalled, err := recaller.RecallPage(context.Background(), requestedURL)
	if err != nil {
		t.Fatalf("recall page: %v", err)
	}
	if recalled.Markdown != recalledMarkdown {
		t.Errorf("markdown = %q", recalled.Markdown)
	}
}

func TestRecallPageFailsWhenTheCrawlOutcomeCannotBeRead(t *testing.T) {
	recaller := recallerBuilder{
		crawlOutcomes: &crawlOutcomes{failWith: errors.New("crawler down")},
	}.recaller()

	if _, err := recaller.RecallPage(context.Background(), requestedURL); err == nil {
		t.Fatal("expected a crawl outcome error")
	}
}

func TestRecallPageFailsWhenTheCrawlOrderCannotBePlaced(t *testing.T) {
	recaller := recallerBuilder{
		crawlOrders: &crawlOrders{failWith: errors.New("no stream")},
	}.recaller()

	if _, err := recaller.RecallPage(context.Background(), requestedURL); err == nil {
		t.Fatal("expected a crawl order placement error")
	}
}

func TestRecallPageFailsWhenTheCorpusReadFails(t *testing.T) {
	recaller := recallerBuilder{
		markdownCorpus: &markdownCorpus{failWith: errors.New("corpus down")},
	}.recaller()

	_, err := recaller.RecallPage(context.Background(), requestedURL)
	if err == nil || errors.Is(err, markdownrecall.ErrMarkdownUnavailable) {
		t.Fatalf("err = %v, want a corpus read failure", err)
	}
}

func TestRecallPageRejectsARecallBeyondTheInFlightLimit(t *testing.T) {
	blocking := &blockingCrawlOrders{entered: make(chan struct{}), release: make(chan struct{})}
	recaller := recallerBuilder{crawlOrders: blocking, maxInFlight: 1}.recaller()

	go func() { _, _ = recaller.RecallPage(context.Background(), requestedURL) }()
	<-blocking.entered

	_, err := recaller.RecallPage(context.Background(), requestedURL)

	if !errors.Is(err, markdownrecall.ErrTooManyRecallsInFlight) {
		t.Fatalf("err = %v, want ErrTooManyRecallsInFlight", err)
	}
	close(blocking.release)
}
