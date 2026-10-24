// Package pagehtmlreading reads what a fetched HTML page states in its markup: the
// refusals of its robots meta tags and the URLs it links to.
package pagehtmlreading

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
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
	Refusals   robotsmeta.Refusals
	LinkedURLs []canonicalurl.CanonicalURL
}

type HTMLPageReading struct {
	htmlParser    HTMLParser
	linkDiscovery LinkDiscovery
}

func NewHTMLPageReading(htmlParser HTMLParser, linkDiscovery LinkDiscovery) *HTMLPageReading {
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
	return Reading{
		Refusals:   htmlmeta.RefusalsOf(page.ContentType, page.Body),
		LinkedURLs: reading.linkDiscovery.LinkedURLsFrom(ctx, elementTree, pageURL),
	}, nil
}
