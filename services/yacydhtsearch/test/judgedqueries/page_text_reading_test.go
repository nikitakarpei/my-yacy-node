package judgedqueries_test

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type pageTextReading struct {
	pageFetch         pagefetch.Fetcher
	formatDerivations pageformats.FormatDerivationCatalog
	pageReadBudget    time.Duration
}

func pageTextReadingOverTheWeb(t *testing.T) pageTextReading {
	t.Helper()

	return pageTextReading{
		pageFetch: pagefetchershttp.New(
			nil,
			pagefetchershttp.ProxyDialTunnel,
			pageFetchUserAgent,
			pageByteCeiling,
			pageReadBudget,
		),
		formatDerivations: formatDerivationsOfThePages(t),
		pageReadBudget:    pageReadBudget,
	}
}

func formatDerivationsOfThePages(t *testing.T) pageformats.FormatDerivationCatalog {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page format derivations: %v", err)
	}

	return formatDerivations
}

func (r pageTextReading) pageTextPerDocument(
	ctx context.Context,
	answeredItems []peeranswers.AnsweredItem,
) map[yacymodel.URLHash]string {
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, r.pageReadBudget)
	defer stopPageReadBudget()

	pageTextOfEachPlace := make([]string, len(answeredItems))
	var pagesBeingRead sync.WaitGroup
	for place, answeredItem := range answeredItems {
		pagesBeingRead.Add(1)
		go func() {
			defer pagesBeingRead.Done()
			pageTextOfEachPlace[place] = r.pageTextOf(budgetedCtx, answeredItem.Metadata.Address)
		}()
	}
	pagesBeingRead.Wait()

	return pageTextPerDocumentOf(answeredItems, pageTextOfEachPlace)
}

func (r pageTextReading) pageTextOf(ctx context.Context, address string) string {
	pageURL, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		return ""
	}
	fetched, err := r.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil || fetched.Status != pagefetch.FetchSucceeded {
		return ""
	}
	document, err := documentextraction.DocumentFrom(
		ctx, fetched.Page.Body, fetched.Page.ContentType, pageURL,
	)
	if err != nil {
		return ""
	}

	return r.textOfTheDocument(ctx, document, pageURL)
}

func (r pageTextReading) textOfTheDocument(
	ctx context.Context,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) string {
	readableText, readableTextDerived := r.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, document, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		return string(readableText)
	}
	fullText, fullTextDerived := r.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, document, pageURL,
	)
	if !fullTextDerived {
		return ""
	}

	return string(bytes.TrimSpace(fullText))
}

func pageTextPerDocumentOf(
	answeredItems []peeranswers.AnsweredItem,
	pageTextOfEachPlace []string,
) map[yacymodel.URLHash]string {
	pageTextPerDocument := make(map[yacymodel.URLHash]string, len(answeredItems))
	for place, answeredItem := range answeredItems {
		if pageTextOfEachPlace[place] == "" {
			continue
		}
		pageTextPerDocument[answeredItem.Metadata.Hash] = pageTextOfEachPlace[place]
	}

	return pageTextPerDocument
}
