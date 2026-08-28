// Package pagehtmlreading reads a fetched page as HTML: the refusals it states,
// and the URLs it links to when it does not refuse link discovery.
package pagehtmlreading

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

var ErrPageNotHTML = errors.New("page is not html")

type Reading struct {
	Refusals       pagerefusals.Refusals
	DiscoveredURLs []canonicalurl.CanonicalURL
}

func ReadingOfPage(
	ctx context.Context,
	page pagefetch.FetchedPage,
	ignored pagerefusals.IgnoredRefusals,
) (Reading, error) {
	elementTree, err := pagehtml.ElementTreeFrom(ctx, page.ContentType, page.Body)
	if err != nil {
		return Reading{}, fmt.Errorf("%w: %w", ErrPageNotHTML, err)
	}
	refusals := pagerefusals.RefusalsOfPage(page.RobotsDirectives, elementTree).
		HonoredBy(ignored)
	return Reading{
		Refusals:       refusals,
		DiscoveredURLs: discoveredURLsFrom(ctx, elementTree, page.LandedURL, refusals),
	}, nil
}

func discoveredURLsFrom(
	ctx context.Context,
	elementTree pagehtml.ElementTree,
	landedURL canonicalurl.CanonicalURL,
	refusals pagerefusals.Refusals,
) []canonicalurl.CanonicalURL {
	if refusals.RefusesLinkDiscovery {
		return nil
	}
	return linkdiscovery.LinkedURLsFrom(ctx, elementTree, landedURL)
}
