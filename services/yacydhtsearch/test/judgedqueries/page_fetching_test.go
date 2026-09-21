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

const (
	budgetPerPage      = 10 * time.Second
	pageByteCeiling    = 4 * 1024 * 1024
	pageFetchUserAgent = "yacydhtsearch (+https://yacy.net)"
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
			budgetPerPage,
		),
		pagesBudget: pagesBudget,
	}
}

func (fetching pageFetching) fetchedPagesOf(
	ctx context.Context,
	foundDocuments []queryanswers.FoundDocument,
) []storedPage {
	budgetedCtx, stopPagesBudget := context.WithTimeout(ctx, fetching.pagesBudget)
	defer stopPagesBudget()

	pagePerPlace := make([]storedPage, len(foundDocuments))
	var pagesBeingFetched sync.WaitGroup
	for place, foundDocument := range foundDocuments {
		pagesBeingFetched.Add(1)
		go func() {
			defer pagesBeingFetched.Done()
			pagePerPlace[place] = fetching.fetchedPageAt(budgetedCtx, foundDocument.Address)
		}()
	}
	pagesBeingFetched.Wait()

	return pagesWithBodyAmong(pagePerPlace)
}

func (fetching pageFetching) fetchedPageAt(ctx context.Context, address string) storedPage {
	pageURL, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		return storedPage{}
	}
	fetchOutcome, err := fetching.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil || fetchOutcome.Status != pagefetch.FetchSucceeded {
		return storedPage{}
	}

	return storedPage{
		address:     address,
		contentType: fetchOutcome.Page.ContentType,
		body:        fetchOutcome.Page.Body,
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
