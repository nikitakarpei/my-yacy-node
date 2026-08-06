// Package redirectrecording records where a fetch's redirect chain landed, then returns the fetch.
package redirectrecording

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	msgRedirectURLRejected  = "redirect chain url rejected"
	msgRedirectRecordFailed = "redirect resolution record failed"
)

type RedirectResolutions interface {
	Record(ctx context.Context, requested, canonical string) error
}

type Fetch struct {
	resolutions RedirectResolutions
	fetcher     pagevisit.Fetcher
}

func New(resolutions RedirectResolutions, fetcher pagevisit.Fetcher) *Fetch {
	return &Fetch{resolutions: resolutions, fetcher: fetcher}
}

func (f *Fetch) Fetch(
	ctx context.Context,
	rawURL string,
	knownVersion pagevisit.PageVersion,
) (pagevisit.FetchOutcome, error) {
	outcome, err := f.fetcher.Fetch(ctx, rawURL, knownVersion)
	if err != nil {
		return pagevisit.FetchOutcome{}, err
	}
	f.record(ctx, outcome)
	return outcome, nil
}

func (f *Fetch) record(ctx context.Context, outcome pagevisit.FetchOutcome) {
	if len(outcome.RedirectChain) == 0 || outcome.Page.FinalURL == "" {
		return
	}
	canonicalFinal, err := canonicalurl.Canonicalize(outcome.Page.FinalURL)
	if err != nil {
		slog.WarnContext(ctx, msgRedirectURLRejected,
			slog.String("url", outcome.Page.FinalURL),
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
		if err := f.resolutions.Record(ctx, canonicalHop, canonicalFinal); err != nil {
			slog.WarnContext(ctx, msgRedirectRecordFailed,
				slog.String("requested", canonicalHop),
				slog.String("canonical", canonicalFinal),
				slog.Any("error", err),
			)
		}
	}
}
