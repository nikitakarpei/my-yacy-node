// Package pagereading reads what a fetched page states, whatever its format: the refusals
// of its headers and of its format, and the URLs it links to when it does not refuse link
// discovery. Only an HTML page links anywhere. A refused link discovery goes to the observer.
package pagereading

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
)

type HTMLPageReading interface {
	ReadingOfPage(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		page pagefetch.FetchedPage,
	) (pagehtmlreading.Reading, error)
}

type Reading struct {
	Refusals       robotsmeta.Refusals
	DiscoveredURLs []canonicalurl.CanonicalURL
}

type PageReading struct {
	htmlPageReading            HTMLPageReading
	refusalEnforcementObserver RefusalEnforcementObserver
}

func NewPageReading(
	htmlPageReading HTMLPageReading,
	refusalEnforcementObserver RefusalEnforcementObserver,
) *PageReading {
	return &PageReading{
		htmlPageReading:            htmlPageReading,
		refusalEnforcementObserver: refusalEnforcementObserver,
	}
}

func (reading *PageReading) ReadingOfPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	page pagefetch.FetchedPage,
) (Reading, error) {
	headerRefusals := httpheader.RefusalsOf(page.RobotsTagValues)
	htmlReading, err := reading.htmlPageReading.ReadingOfPage(ctx, pageURL, page)
	if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		return Reading{Refusals: headerRefusals}, nil
	}
	if err != nil {
		return Reading{}, err
	}
	refusals := headerRefusals.With(htmlReading.Refusals)
	return Reading{
		Refusals: refusals,
		DiscoveredURLs: reading.discoveredURLsFrom(
			ctx, pageURL, htmlReading.LinkedURLs, refusals,
		),
	}, nil
}

func (reading *PageReading) discoveredURLsFrom(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	linkedURLs []canonicalurl.CanonicalURL,
	refusals robotsmeta.Refusals,
) []canonicalurl.CanonicalURL {
	if refusals.RefusesLinkDiscovery {
		reading.refusalEnforcementObserver.LinkDiscoveryRefusalEnforced(ctx, pageURL)
		return nil
	}
	return linkedURLs
}
