// Package pagehtmlreading reads a fetched page as HTML: the refusals it states,
// and the URLs it links to when it does not refuse link discovery.
package pagehtmlreading

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

var ErrPageNotHTML = errors.New("page is not html")

type HTMLParser interface {
	ElementTreeFrom(
		ctx context.Context,
		contentType string,
		body []byte,
	) (pagehtml.ElementTree, error)
}

type LinkDiscovery interface {
	LinkedURLsFrom(
		ctx context.Context,
		elementTree pagehtml.ElementTree,
		pageURL canonicalurl.CanonicalURL,
	) []canonicalurl.CanonicalURL
}

type Reading struct {
	Refusals       pagerefusals.Refusals
	DiscoveredURLs []canonicalurl.CanonicalURL
}

type HTMLPageReading struct {
	htmlParser    HTMLParser
	linkDiscovery LinkDiscovery
}

func NewHTMLPageReading(
	htmlParser HTMLParser,
	linkDiscovery LinkDiscovery,
) *HTMLPageReading {
	return &HTMLPageReading{htmlParser: htmlParser, linkDiscovery: linkDiscovery}
}

func (reading *HTMLPageReading) ReadingOfPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	page pagefetch.FetchedPage,
) (Reading, error) {
	elementTree, err := reading.htmlParser.ElementTreeFrom(ctx, page.ContentType, page.Body)
	if errors.Is(err, pagehtml.ErrNotHTML) {
		return Reading{}, fmt.Errorf("%w: %w", ErrPageNotHTML, err)
	}
	if err != nil {
		return Reading{}, err
	}
	refusals := pagerefusals.RefusalsOfPage(page.RobotsDirectives, elementTree)
	return Reading{
		Refusals: refusals,
		DiscoveredURLs: reading.discoveredURLsFrom(
			ctx, elementTree, pageURL, refusals,
		),
	}, nil
}

func (reading *HTMLPageReading) discoveredURLsFrom(
	ctx context.Context,
	elementTree pagehtml.ElementTree,
	pageURL canonicalurl.CanonicalURL,
	refusals pagerefusals.Refusals,
) []canonicalurl.CanonicalURL {
	if refusals.RefusesLinkDiscovery {
		return nil
	}
	return reading.linkDiscovery.LinkedURLsFrom(ctx, elementTree, pageURL)
}
