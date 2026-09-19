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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
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

type readPage struct {
	title string
	text  string
}

type readPages struct {
	pageTextPerDocument  map[yacymodel.URLHash]string
	pageTitlePerDocument map[yacymodel.URLHash]string
}

func (r pageTextReading) readPagesOf(
	ctx context.Context,
	foundDocuments []queryanswers.FoundDocument,
) readPages {
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, r.pageReadBudget)
	defer stopPageReadBudget()

	readPageOfEachPlace := make([]readPage, len(foundDocuments))
	var pagesBeingRead sync.WaitGroup
	for place, foundDocument := range foundDocuments {
		pagesBeingRead.Add(1)
		go func() {
			defer pagesBeingRead.Done()
			readPageOfEachPlace[place] = r.readPageOf(budgetedCtx, foundDocument.Address)
		}()
	}
	pagesBeingRead.Wait()

	return readPagesFrom(foundDocuments, readPageOfEachPlace)
}

func (r pageTextReading) readPageOf(ctx context.Context, address string) readPage {
	pageURL, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		return readPage{}
	}
	fetched, err := r.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil || fetched.Status != pagefetch.FetchSucceeded {
		return readPage{}
	}
	document, err := documentextraction.DocumentFrom(
		ctx, fetched.Page.Body, fetched.Page.ContentType, pageURL,
	)
	if err != nil {
		return readPage{}
	}

	return readPage{title: document.Title, text: r.textOfTheDocument(ctx, document, pageURL)}
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

func readPagesFrom(
	foundDocuments []queryanswers.FoundDocument,
	readPageOfEachPlace []readPage,
) readPages {
	pages := readPages{
		pageTextPerDocument:  make(map[yacymodel.URLHash]string, len(foundDocuments)),
		pageTitlePerDocument: make(map[yacymodel.URLHash]string, len(foundDocuments)),
	}
	for place, foundDocument := range foundDocuments {
		if readPageOfEachPlace[place].text == "" {
			continue
		}
		pages.pageTextPerDocument[foundDocument.Hash] = readPageOfEachPlace[place].text
		if readPageOfEachPlace[place].title == "" {
			continue
		}
		pages.pageTitlePerDocument[foundDocument.Hash] = readPageOfEachPlace[place].title
	}

	return pages
}
