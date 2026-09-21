package judgedqueries_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const snippetLengthCeiling = 300

type answersAndPageContents struct {
	answers                 queryanswers.AnsweredQuery
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents
}

func answersAndPageContentsOf(
	answers queryanswers.AnsweredQuery,
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) answersAndPageContents {
	return answersAndPageContents{
		answers:                 answers.WithReadPages(pageContentsPerDocument),
		pageContentsPerDocument: pageContentsPerDocument,
	}
}

type pageExtraction struct {
	formatDerivations pageformats.FormatDerivationCatalog
}

func pageExtractionOfEveryFormat(t *testing.T) pageExtraction {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page format derivations: %v", err)
	}

	return pageExtraction{formatDerivations: formatDerivations}
}

func (extraction pageExtraction) pageContentsPerDocumentOf(
	ctx context.Context,
	answers queryanswers.AnsweredQuery,
	storedPagePerAddress map[string]storedPage,
) map[yacymodel.URLHash]pagecontents.PageContents {
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, foundDocument := range answers.FoundDocuments {
		page, stored := storedPagePerAddress[foundDocument.Address]
		if !stored {
			continue
		}
		extractedPage := extraction.extractedPageOf(ctx, page)
		if extractedPage.text == "" {
			continue
		}
		pageContentsPerDocument[foundDocument.Hash] = pagecontents.PageContentsFrom(
			extractedPage.title,
			extractedPage.text,
			extractedPage.linkCounts,
			answers.QueryWords,
			snippetLengthCeiling,
		)
	}

	return pageContentsPerDocument
}

type extractedPage struct {
	title      string
	text       string
	linkCounts pagecontents.LinkCounts
}

func (extraction pageExtraction) extractedPageOf(
	ctx context.Context, page storedPage,
) extractedPage {
	pageURL, err := canonicalurl.CanonicalURLOf(page.address)
	if err != nil {
		return extractedPage{}
	}
	document, err := documentextraction.DocumentFrom(ctx, page.body, page.contentType, pageURL)
	if err != nil {
		return extractedPage{}
	}

	return extractedPage{
		title: document.Title,
		text:  extraction.textOf(ctx, document, pageURL),
		linkCounts: pagecontents.LinkCounts{
			LocalLinks:    document.LocalLinks,
			ExternalLinks: document.ExternalLinks,
		},
	}
}

func (extraction pageExtraction) textOf(
	ctx context.Context,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) string {
	readableText, readableTextDerived := extraction.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, document, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		return string(readableText)
	}
	fullText, fullTextDerived := extraction.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, document, pageURL,
	)
	if !fullTextDerived {
		return ""
	}

	return string(bytes.TrimSpace(fullText))
}
