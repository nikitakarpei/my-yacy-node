package pageabsorption_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
)

type fakeExtract struct {
	document contentextraction.ExtractedDocument
	err      error
}

func (f fakeExtract) DocumentFrom(
	_ context.Context,
	_, _ string,
	_ []byte,
) (contentextraction.ExtractedDocument, error) {
	return f.document, f.err
}

func document(title, text string) contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Title:  title,
		Body:   []byte(text),
		Format: contentformatgraph.FormatReadableText,
	}
}

func refusingDocument() contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Body:            []byte("b"),
		Format:          contentformatgraph.FormatReadableText,
		RefusesIndexing: true,
	}
}

func linkingDocument(t *testing.T, discovered string) contentextraction.ExtractedDocument {
	t.Helper()
	return contentextraction.ExtractedDocument{
		Body:   []byte("b"),
		Format: contentformatgraph.FormatReadableText,
		DiscoveredURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, discovered),
		},
	}
}

func newAbsorber(extractor pageabsorption.PageExtractor) pageabsorption.Absorber {
	return pageabsorption.New(extractor).AbsorberFor(pageabsorption.Honored)
}

func succeeded(finalURL string) pagefetch.FetchedPage {
	return pagefetch.FetchedPage{
		FinalURL:    finalURL,
		ContentType: "text/html",
		Body:        []byte("x"),
	}
}
