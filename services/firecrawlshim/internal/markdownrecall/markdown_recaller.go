// Package markdownrecall yields the markdown an operator's corpus holds for a URL. It
// orders a crawl of the URL and waits until the corpus holds the markdown, crawling
// disposes of the page, or the recall limit runs out.
package markdownrecall

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
)

var (
	ErrTooManyRecallsInFlight = errors.New("too many recalls in flight")
	ErrMarkdownUnavailable    = errors.New("the corpus holds no markdown for the page")
)

type CrawlOutcomes interface {
	ReadPage(
		ctx context.Context,
		in *crawlerv1.ReadPageRequest,
		opts ...grpc.CallOption,
	) (*crawlerv1.PageOutcome, error)
}

type CrawlOrders interface {
	Place(ctx context.Context, canonicalURL string) error
}

type MarkdownCorpus interface {
	RecallPage(
		ctx context.Context,
		in *corpusmarkdownv1.RecallPageRequest,
		opts ...grpc.CallOption,
	) (*corpusmarkdownv1.RecallPageResponse, error)
}

type Config struct {
	RecallLimit        time.Duration
	PollInterval       time.Duration
	MaxRecallsInFlight int
}

type RecalledPage struct {
	CanonicalURL string
	Markdown     string
}

type MarkdownRecaller struct {
	crawlOutcomes   CrawlOutcomes
	crawlOrders     CrawlOrders
	markdownCorpus  MarkdownCorpus
	recallLimit     time.Duration
	pollInterval    time.Duration
	recallsInFlight chan struct{}
}

func NewMarkdownRecaller(
	crawlOutcomes CrawlOutcomes,
	crawlOrders CrawlOrders,
	markdownCorpus MarkdownCorpus,
	config Config,
) *MarkdownRecaller {
	return &MarkdownRecaller{
		crawlOutcomes:   crawlOutcomes,
		crawlOrders:     crawlOrders,
		markdownCorpus:  markdownCorpus,
		recallLimit:     config.RecallLimit,
		pollInterval:    config.PollInterval,
		recallsInFlight: make(chan struct{}, config.MaxRecallsInFlight),
	}
}

func (r *MarkdownRecaller) RecallPage(
	ctx context.Context,
	requestedURL string,
) (RecalledPage, error) {
	select {
	case r.recallsInFlight <- struct{}{}:
		defer func() { <-r.recallsInFlight }()
	default:
		return RecalledPage{}, ErrTooManyRecallsInFlight
	}

	outcome, err := r.crawlOutcomes.ReadPage(ctx, &crawlerv1.ReadPageRequest{Url: requestedURL})
	if err != nil {
		return RecalledPage{}, fmt.Errorf("read crawl outcome of %q: %w", requestedURL, err)
	}
	if err := r.crawlOrders.Place(ctx, outcome.GetCanonicalUrl()); err != nil {
		return RecalledPage{}, fmt.Errorf(
			"place crawl order for %q: %w", outcome.GetCanonicalUrl(), err,
		)
	}

	recallCtx, cancel := context.WithTimeout(ctx, r.recallLimit)
	defer cancel()
	recalled, err := r.markdownAwaitedFor(recallCtx, outcome)
	if err != nil && !errors.Is(err, ErrMarkdownUnavailable) && recallCtx.Err() != nil {
		return RecalledPage{}, fmt.Errorf("%w within the recall limit", ErrMarkdownUnavailable)
	}
	return recalled, err
}

func (r *MarkdownRecaller) markdownAwaitedFor(
	ctx context.Context,
	outcome *crawlerv1.PageOutcome,
) (RecalledPage, error) {
	disposalMarkAtOrder := outcome.GetDisposal().GetMark()
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		recalled, held, err := r.markdownHeldFor(ctx, outcome.GetResolvedUrl())
		if err != nil {
			return RecalledPage{}, err
		}
		if held {
			return recalled, nil
		}
		outcome, err = r.crawlOutcomes.ReadPage(
			ctx, &crawlerv1.ReadPageRequest{Url: outcome.GetCanonicalUrl()},
		)
		if err != nil {
			return RecalledPage{}, fmt.Errorf("read crawl outcome: %w", err)
		}
		if disposal := outcome.GetDisposal(); disposal.GetMark() != disposalMarkAtOrder {
			return RecalledPage{}, fmt.Errorf(
				"%w: crawling disposed of it: %s", ErrMarkdownUnavailable, disposal.GetReason(),
			)
		}
		select {
		case <-ctx.Done():
			return RecalledPage{}, fmt.Errorf(
				"%w within the recall limit", ErrMarkdownUnavailable,
			)
		case <-ticker.C:
		}
	}
}

func (r *MarkdownRecaller) markdownHeldFor(
	ctx context.Context,
	resolvedURL string,
) (RecalledPage, bool, error) {
	recalled, err := r.markdownCorpus.RecallPage(
		ctx, &corpusmarkdownv1.RecallPageRequest{Url: resolvedURL},
	)
	switch {
	case status.Code(err) == codes.NotFound:
		return RecalledPage{}, false, nil
	case err != nil:
		return RecalledPage{}, false, fmt.Errorf("recall markdown of %q: %w", resolvedURL, err)
	}
	return RecalledPage{
		CanonicalURL: recalled.GetCanonicalUrl(),
		Markdown:     recalled.GetMarkdown(),
	}, true, nil
}
