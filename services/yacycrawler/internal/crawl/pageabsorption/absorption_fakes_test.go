package pageabsorption_test

import (
	"context"

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

func linkingDocument(discovered string) contentextraction.ExtractedDocument {
	return contentextraction.ExtractedDocument{
		Body:           []byte("b"),
		Format:         contentformatgraph.FormatReadableText,
		DiscoveredURLs: []string{discovered},
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
