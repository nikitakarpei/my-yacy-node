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
	pageFetch   pagefetch.Fetcher
	pagesBudget time.Duration
}

func pageFetchingOverTheWeb(pagesBudget time.Duration) pageFetching {
	return pageFetching{
		pageFetch: pagefetchershttp.New(
			nil,
			pagefetchershttp.ProxyDialTunnel,
			pageFetchUserAgent,
			pageByteCeiling,
			pageReadBudget,
		),
		pagesBudget: pagesBudget,
	}
}

func (f pageFetching) fetchedPagesOf(
	ctx context.Context,
	foundDocuments []queryanswers.FoundDocument,
) []storedPage {
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, f.pagesBudget)
	defer stopPageReadBudget()

	pageOfEachPlace := make([]storedPage, len(foundDocuments))
	var pagesBeingRead sync.WaitGroup
	for place, foundDocument := range foundDocuments {
		pagesBeingRead.Add(1)
		go func() {
			defer pagesBeingRead.Done()
			pageOfEachPlace[place] = f.fetchedPageOf(budgetedCtx, foundDocument.Address)
		}()
	}
	pagesBeingRead.Wait()

	return pagesThatCarryABody(pageOfEachPlace)
}

func (f pageFetching) fetchedPageOf(ctx context.Context, address string) storedPage {
	pageURL, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		return storedPage{}
	}
	fetched, err := f.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil || fetched.Status != pagefetch.FetchSucceeded {
		return storedPage{}
	}

	return storedPage{
		address:     address,
		contentType: fetched.Page.ContentType,
		body:        fetched.Page.Body,
	}
}

func pagesThatCarryABody(pageOfEachPlace []storedPage) []storedPage {
	pages := make([]storedPage, 0, len(pageOfEachPlace))
	for _, page := range pageOfEachPlace {
		if len(page.body) == 0 {
			continue
		}
		pages = append(pages, page)
	}

	return pages
}
