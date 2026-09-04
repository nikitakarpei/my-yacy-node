// Package pageread answers a call for one page with the markdown the operator's own corpus
// holds for it, and asks the stack to fetch the page first when the corpus holds nothing
// the call accepts. It waits for that fetch only as long as the call may wait, and says
// what became of it.
package pageread

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdownexcerpt"
)

var ErrPageNotInCorpus = errors.New("the corpus holds no markdown for the page")

type PageMarkdown struct {
	Markdown string
	Version  string
	StoredAt time.Time
}

type MarkdownCorpus interface {
	PageMarkdownAt(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	) (PageMarkdown, error)
}

type ScrapeRequests interface {
	AskToScrape(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type ScrapeOutcomes interface {
	ListenerFor(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	) (ScrapeOutcomeListener, error)
}

type ScrapeOutcomeListener interface {
	AwaitedFetchOutcome(ctx context.Context) FetchOutcome
	Close()
}

type PageCall struct {
	URL            canonicalurl.CanonicalURL
	CharacterLimit int
	ToleratedAge   time.Duration
	Version        string
}

type PageAnswer struct {
	URL          canonicalurl.CanonicalURL
	Version      string
	StoredAt     time.Time
	FetchOutcome FetchOutcome
	Excerpt      markdownexcerpt.MarkdownExcerpt
}

type Config struct {
	Corpus          MarkdownCorpus
	ScrapeRequests  ScrapeRequests
	ScrapeOutcomes  ScrapeOutcomes
	Progress        PageReadProgress
	CharacterLimit  int
	ScrapeTolerance time.Duration
	FetchWait       time.Duration
}

type PageReader struct {
	corpus          MarkdownCorpus
	scrapeRequests  ScrapeRequests
	scrapeOutcomes  ScrapeOutcomes
	progress        PageReadProgress
	characterLimit  int
	scrapeTolerance time.Duration
	fetchWait       time.Duration
}

func NewPageReader(cfg Config) *PageReader {
	return &PageReader{
		corpus:          cfg.Corpus,
		scrapeRequests:  cfg.ScrapeRequests,
		scrapeOutcomes:  cfg.ScrapeOutcomes,
		progress:        cfg.Progress,
		characterLimit:  cfg.CharacterLimit,
		scrapeTolerance: cfg.ScrapeTolerance,
		fetchWait:       cfg.FetchWait,
	}
}

func (r *PageReader) PageAnswerFor(ctx context.Context, call PageCall) (PageAnswer, error) {
	stored, err := r.storedPageMarkdownAt(ctx, call.URL)
	if err != nil {
		return PageAnswer{}, err
	}
	if r.fetchIsNotNeeded(call, stored) {
		r.progress.PageAnswered(ctx, call.URL, FetchNotNeeded)
		return r.pageAnswerFrom(call, stored, FetchNotNeeded), nil
	}

	fetchOutcome := r.fetchPage(ctx, call.URL)
	if fetchOutcome == PageFetched {
		stored, err = r.storedPageMarkdownAt(ctx, call.URL)
		if err != nil {
			return PageAnswer{}, err
		}
	}
	r.progress.PageAnswered(ctx, call.URL, fetchOutcome)
	return r.pageAnswerFrom(call, stored, fetchOutcome), nil
}

func (r *PageReader) storedPageMarkdownAt(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) (*PageMarkdown, error) {
	stored, err := r.corpus.PageMarkdownAt(ctx, pageURL)
	if errors.Is(err, ErrPageNotInCorpus) {
		return nil, nil
	}
	if err != nil {
		r.progress.MarkdownRecallFailed(ctx, pageURL, err)
		return nil, fmt.Errorf("recall %q from the corpus: %w", pageURL, err)
	}
	return &stored, nil
}

func (r *PageReader) fetchIsNotNeeded(call PageCall, stored *PageMarkdown) bool {
	if stored == nil {
		return false
	}
	if call.Version != "" {
		return true
	}
	return time.Since(stored.StoredAt) <= r.toleratedAgeFor(call)
}

func (r *PageReader) toleratedAgeFor(call PageCall) time.Duration {
	if call.ToleratedAge > r.scrapeTolerance {
		return call.ToleratedAge
	}
	return r.scrapeTolerance
}

func (r *PageReader) pageAnswerFrom(
	call PageCall,
	stored *PageMarkdown,
	fetchOutcome FetchOutcome,
) PageAnswer {
	if stored == nil {
		stored = &PageMarkdown{}
	}
	return PageAnswer{
		URL:          call.URL,
		Version:      stored.Version,
		StoredAt:     stored.StoredAt,
		FetchOutcome: fetchOutcome,
		Excerpt: markdownexcerpt.MarkdownExcerptOf(
			stored.Markdown,
			r.characterLimitFor(call),
		),
	}
}

func (r *PageReader) characterLimitFor(call PageCall) int {
	if call.CharacterLimit > 0 {
		return call.CharacterLimit
	}
	return r.characterLimit
}

func (r *PageReader) fetchPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) FetchOutcome {
	fetchCtx, stopWaiting := context.WithTimeout(ctx, r.fetchWait)
	defer stopWaiting()

	listener, err := r.scrapeOutcomes.ListenerFor(fetchCtx, pageURL)
	if err != nil {
		return FetchUnfinished
	}
	defer listener.Close()

	r.scrapeRequests.AskToScrape(fetchCtx, pageURL)
	fetchOutcome := listener.AwaitedFetchOutcome(fetchCtx)
	if fetchOutcome == FetchUnfinished {
		r.progress.FetchOutcomeNotHeard(ctx, pageURL, r.fetchWait)
	}
	return fetchOutcome
}
