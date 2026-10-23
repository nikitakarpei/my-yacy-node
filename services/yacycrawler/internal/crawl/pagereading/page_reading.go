// Package pagereading reads a fetched page: the refusals it states, and the URLs it
// links to when it is HTML and does not refuse link discovery. A page that is not HTML
// states its refusals only in its headers. A refused link discovery goes to the observer.
package pagereading

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

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
	Refusals       robotsmeta.Refusals
	DiscoveredURLs []canonicalurl.CanonicalURL
}

type PageReading struct {
	htmlParser                 HTMLParser
	linkDiscovery              LinkDiscovery
	refusalEnforcementObserver RefusalEnforcementObserver
}

func NewPageReading(
	htmlParser HTMLParser,
	linkDiscovery LinkDiscovery,
	refusalEnforcementObserver RefusalEnforcementObserver,
) *PageReading {
	return &PageReading{
		htmlParser:                 htmlParser,
		linkDiscovery:              linkDiscovery,
		refusalEnforcementObserver: refusalEnforcementObserver,
	}
}

func (reading *PageReading) ReadingOfPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	page pagefetch.FetchedPage,
) (Reading, error) {
	elementTree, err := reading.htmlParser.ElementTreeFrom(ctx, page.ContentType, page.Body)
	if errors.Is(err, pagehtml.ErrNotHTML) {
		return Reading{Refusals: refusalsOf(page)}, nil
	}
	if err != nil {
		return Reading{}, err
	}
	refusals := refusalsOf(page)
	return Reading{
		Refusals: refusals,
		DiscoveredURLs: reading.discoveredURLsFrom(
			ctx, elementTree, pageURL, refusals,
		),
	}, nil
}

func refusalsOf(page pagefetch.FetchedPage) robotsmeta.Refusals {
	return httpheader.RefusalsOf(page.RobotsTagValues).
		With(htmlmeta.RefusalsOf(page.ContentType, page.Body))
}

func (reading *PageReading) discoveredURLsFrom(
	ctx context.Context,
	elementTree pagehtml.ElementTree,
	pageURL canonicalurl.CanonicalURL,
	refusals robotsmeta.Refusals,
) []canonicalurl.CanonicalURL {
	if refusals.RefusesLinkDiscovery {
		reading.refusalEnforcementObserver.LinkDiscoveryRefusalEnforced(ctx, pageURL)
		return nil
	}
	return reading.linkDiscovery.LinkedURLsFrom(ctx, elementTree, pageURL)
}
