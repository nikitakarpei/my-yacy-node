package judgedqueries_test

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

type pageFetching struct {
	fetcher     pagefetch.Fetcher
	pagesBudget time.Duration
}

func pageFetchingWithin(pagesBudget time.Duration) pageFetching {
	return pageFetching{
		fetcher: pagefetchershttp.New(
			nil,
			pagefetchershttp.ProxyDialTunnel,
			pageFetchUserAgent,
			pageByteCeiling,
			pageBudget,
		),
		pagesBudget: pagesBudget,
	}
}

func (fetching pageFetching) fetchedPagesOf(
	ctx context.Context,
	foundDocuments []queryanswers.FoundDocument,
) []storedPage {
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, fetching.pagesBudget)
	defer stopPageReadBudget()

	pagePerPlace := make([]storedPage, len(foundDocuments))
	var pagesBeingRead sync.WaitGroup
	for place, foundDocument := range foundDocuments {
		pagesBeingRead.Add(1)
		go func() {
			defer pagesBeingRead.Done()
			pagePerPlace[place] = fetching.fetchedPageAt(budgetedCtx, foundDocument.Address)
		}()
	}
	pagesBeingRead.Wait()

	return pagesWithBodyAmong(pagePerPlace)
}

func (fetching pageFetching) fetchedPageAt(ctx context.Context, address string) storedPage {
	pageURL, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		return storedPage{}
	}
	fetched, err := fetching.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil || fetched.Status != pagefetch.FetchSucceeded {
		return storedPage{}
	}

	return storedPage{
		address:     address,
		contentType: fetched.Page.ContentType,
		body:        fetched.Page.Body,
	}
}

func pagesWithBodyAmong(pagePerPlace []storedPage) []storedPage {
	pages := make([]storedPage, 0, len(pagePerPlace))
	for _, page := range pagePerPlace {
		if len(page.body) == 0 {
			continue
		}
		pages = append(pages, page)
	}

	return pages
}
