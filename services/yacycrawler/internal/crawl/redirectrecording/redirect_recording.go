// Package redirectrecording records where a fetched page's redirect chain landed, then absorbs it.
package redirectrecording

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
)

const (
	msgRedirectURLRejected  = "redirect chain url rejected"
	msgRedirectRecordFailed = "redirect resolution record failed"
)

type RedirectResolutions interface {
	Record(ctx context.Context, requested, canonical string) error
}

type PageAbsorber interface {
	Absorb(ctx context.Context, page fetchedpage.Page) (pageabsorption.AbsorptionOutcome, error)
}

type Absorber struct {
	resolutions RedirectResolutions
	absorber    PageAbsorber
}

func New(resolutions RedirectResolutions, absorber PageAbsorber) *Absorber {
	return &Absorber{resolutions: resolutions, absorber: absorber}
}

func (a *Absorber) Absorb(
	ctx context.Context,
	page fetchedpage.Page,
) (pageabsorption.AbsorptionOutcome, error) {
	a.record(ctx, page)
	return a.absorber.Absorb(ctx, page)
}

func (a *Absorber) record(ctx context.Context, page fetchedpage.Page) {
	canonicalFinal, err := canonicalurl.Canonicalize(page.FinalURL)
	if err != nil {
		slog.WarnContext(ctx, msgRedirectURLRejected,
			slog.String("url", page.FinalURL),
			slog.Any("error", err),
		)
		return
	}
	for _, hop := range page.RedirectChain {
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
		if err := a.resolutions.Record(ctx, canonicalHop, canonicalFinal); err != nil {
			slog.WarnContext(ctx, msgRedirectRecordFailed,
				slog.String("requested", canonicalHop),
				slog.String("canonical", canonicalFinal),
				slog.Any("error", err),
			)
		}
	}
}
