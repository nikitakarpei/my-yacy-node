package judgedqueries_test

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	budgetPerPage      = 10 * time.Second
	pageByteCeiling    = 4 * 1024 * 1024
	pageFetchUserAgent = "yacydhtsearch (+https://yacy.net)"
	maxRedirectHops    = 3
)

type pageFetching struct {
	fetcher     *redirectfollowingfetch.RedirectFollowingPageFetch
	pagesBudget time.Duration
}

func pageFetchingWithin(pagesBudget time.Duration) pageFetching {
	return pageFetching{
		fetcher: redirectfollowingfetch.New(
			pagefetchershttp.New(
				nil,
				pagefetchershttp.ProxyDialTunnel,
				pageFetchUserAgent,
				pageByteCeiling,
				budgetPerPage,
			),
			maxRedirectHops,
		),
		pagesBudget: pagesBudget,
	}
}

func (fetching pageFetching) fetchedPagesOf(
	ctx context.Context,
	foundDocuments []queryfindings.FoundDocument,
) []storedPage {
	return pagesAmong(fetching.pageCapturesOf(ctx, foundDocuments))
}

type pageCapture struct {
	document yacymodel.URLHash
	page     storedPage
	failure  captureFailure
}

type captureFailure string

const (
	passingCaptureFailure captureFailure = "passing"
	refusedCaptureFailure captureFailure = "refused"
	goneCaptureFailure    captureFailure = "gone"
)

func (fetching pageFetching) pageCapturesOf(
	ctx context.Context,
	foundDocuments []queryfindings.FoundDocument,
) []pageCapture {
	budgetedCtx, stopPagesBudget := context.WithTimeout(ctx, fetching.pagesBudget)
	defer stopPagesBudget()

	capturePerPlace := make([]pageCapture, len(foundDocuments))
	var pagesBeingFetched sync.WaitGroup
	for place, foundDocument := range foundDocuments {
		pagesBeingFetched.Add(1)
		go func() {
			defer pagesBeingFetched.Done()
			capturePerPlace[place] = fetching.pageCaptureOf(budgetedCtx, foundDocument)
		}()
	}
	pagesBeingFetched.Wait()

	return capturePerPlace
}

func (fetching pageFetching) pageCaptureOf(
	ctx context.Context, foundDocument queryfindings.FoundDocument,
) pageCapture {
	failed := pageCapture{document: foundDocument.Hash, failure: refusedCaptureFailure}
	pageURL, err := canonicalurl.CanonicalURLOf(foundDocument.Address)
	if err != nil {
		return failed
	}
	landedFetch, err := fetching.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil {
		failed.failure = passingCaptureFailure

		return failed
	}
	if landedFetch.Outcome.Status != pagefetch.FetchSucceeded {
		failed.failure = captureFailureOf(landedFetch.Outcome.Status)

		return failed
	}
	if len(landedFetch.Outcome.Page.Body) == 0 {
		return failed
	}

	return pageCapture{document: foundDocument.Hash, page: storedPage{
		address:     foundDocument.Address,
		contentType: landedFetch.Outcome.Page.ContentType,
		body:        landedFetch.Outcome.Page.Body,
	}}
}

func captureFailureOf(fetchStatus pagefetch.FetchStatus) captureFailure {
	switch fetchStatus {
	case pagefetch.FetchFailed, pagefetch.FetchDeadlinePassed, pagefetch.FetchDeferred:
		return passingCaptureFailure
	case pagefetch.FetchGone:
		return goneCaptureFailure
	default:
		return refusedCaptureFailure
	}
}

func pagesAmong(pageCaptures []pageCapture) []storedPage {
	pages := make([]storedPage, 0, len(pageCaptures))
	for _, capture := range pageCaptures {
		if capture.failure != "" {
			continue
		}
		pages = append(pages, capture.page)
	}

	return pages
}
