// Package scrapedpage holds the page a scrape request read, under the URL that identifies the
// page.
//
// A read that lands at another URL moves the identity of the page only when the request reads
// the page at its own URL. The URL the read landed at stays available for the relative links
// of the page.
package scrapedpage

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

type ScrapedPage struct {
	PageURL     canonicalurl.CanonicalURL
	LandedURL   canonicalurl.CanonicalURL
	ContentType string
	Body        []byte
}

func Of(
	request scraperequestcontract.ScrapeRequest,
	fetchedPage pagefetch.FetchedPage,
) ScrapedPage {
	pageURL := request.PageURL
	if readsThePageAtItsOwnURL(request) {
		pageURL = fetchedPage.LandedURL
	}
	return ScrapedPage{
		PageURL:     pageURL,
		LandedURL:   fetchedPage.LandedURL,
		ContentType: fetchedPage.ContentType,
		Body:        fetchedPage.Body,
	}
}

func readsThePageAtItsOwnURL(request scraperequestcontract.ScrapeRequest) bool {
	return request.FetchURL == request.PageURL
}
