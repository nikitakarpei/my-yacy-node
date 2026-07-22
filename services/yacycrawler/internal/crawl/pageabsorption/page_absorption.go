package pageabsorption

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	msgRedirectURLRejected  = "redirect chain url rejected"
	msgRedirectRecordFailed = "redirect resolution record failed"
)

type Config struct {
	PublishRetryFloor   time.Duration
	PublishRetryCeiling time.Duration
}

type DocumentExtractor interface {
	Extract(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) ([]contentextraction.ExtractedDocument, error)
}

type Absorption struct {
	graph    contentformatgraph.Graph
	extract  DocumentExtractor
	resolve  RedirectResolver
	feeds    []Feed
	observer Progress
	clock    Clock
	config   Config
}

//nolint:revive // argument-limit: the pipeline's collaborators are all distinct ports.
func New(
	graph contentformatgraph.Graph,
	extract DocumentExtractor,
	resolve RedirectResolver,
	feeds []Feed,
	observer Progress,
	clock Clock,
	config Config,
) *Absorption {
	return &Absorption{
		graph:    graph,
		extract:  extract,
		resolve:  resolve,
		feeds:    feeds,
		observer: observer,
		clock:    clock,
		config:   config,
	}
}

func (a *Absorption) Absorb(
	ctx context.Context,
	outcome pagevisit.FetchOutcome,
) ([]string, error) {
	a.recordRedirects(ctx, outcome)
	if outcome.Truncated {
		a.observer.PageDisposed(DisposalOversized)
		return nil, nil
	}
	return a.absorbDocuments(ctx, outcome)
}

func (a *Absorption) recordRedirects(ctx context.Context, outcome pagevisit.FetchOutcome) {
	canonicalFinal, err := canonicalurl.Canonicalize(outcome.FinalURL)
	if err != nil {
		slog.WarnContext(ctx, msgRedirectURLRejected,
			slog.String("url", outcome.FinalURL),
			slog.Any("error", err),
		)
		return
	}
	for _, hop := range outcome.RedirectChain {
		canonicalHop, err := canonicalurl.Canonicalize(hop)
		if err != nil {
			slog.WarnContext(ctx, msgRedirectURLRejected,
				slog.String("url", hop),
				slog.Any("error", err),
			)
			continue
		}
		if canonicalHop == canonicalFinal {
			continue
		}
		if err := a.resolve.Record(ctx, canonicalHop, canonicalFinal); err != nil {
			slog.WarnContext(ctx, msgRedirectRecordFailed,
				slog.String("requested", canonicalHop),
				slog.String("canonical", canonicalFinal),
				slog.Any("error", err),
			)
		}
	}
}
