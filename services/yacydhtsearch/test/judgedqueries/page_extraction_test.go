package judgedqueries_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const snippetLengthCeiling = 300

type findingsAndPageContents struct {
	findings                queryfindings.Findings
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents
}

func findingsAndPageContentsOf(
	findings queryfindings.Findings,
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) findingsAndPageContents {
	return findingsAndPageContents{
		findings:                findings.WithReadPages(pageContentsPerDocument),
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
	queryWords []string,
	findings queryfindings.Findings,
	storedPagePerAddress map[string]storedPage,
) map[yacymodel.URLHash]pagecontents.PageContents {
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, foundDocument := range findings.FoundDocuments {
		page, stored := storedPagePerAddress[foundDocument.Address]
		if !stored {
			continue
		}
		extractedPage := extraction.extractedPageOf(ctx, page)
		if extractedPage.Text == "" {
			continue
		}
		pageContentsPerDocument[foundDocument.Hash] = pagecontents.PageContentsFrom(
			extractedPage,
			queryWords,
			snippetLengthCeiling,
		)
	}

	return pageContentsPerDocument
}

func (extraction pageExtraction) extractedPageOf(
	ctx context.Context, page storedPage,
) pagecontents.ExtractedPage {
	pageURL, err := canonicalurl.CanonicalURLOf(page.address)
	if err != nil {
		return pagecontents.ExtractedPage{}
	}
	document, err := documentextraction.DocumentFrom(ctx, page.body, page.contentType, pageURL)
	if err != nil {
		return pagecontents.ExtractedPage{}
	}
	readableText, readableTextDerived := extraction.formatDerivations.BytesIn(
		ctx, documentextraction.FormatReadableText, document, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		auxiliaryText, _ := extraction.formatDerivations.BytesIn(
			ctx, documentextraction.FormatAuxiliaryText, document, pageURL,
		)
		return extractedPageFrom(document, string(readableText), string(auxiliaryText))
	}
	fullText, _ := extraction.formatDerivations.BytesIn(
		ctx, documentextraction.FormatFullText, document, pageURL,
	)

	return extractedPageFrom(document, string(bytes.TrimSpace(fullText)), "")
}

func extractedPageFrom(
	document documentextraction.Document, text string, auxiliaryText string,
) pagecontents.ExtractedPage {
	return pagecontents.ExtractedPage{
		Title:         document.Title,
		Text:          text,
		AuxiliaryText: auxiliaryText,
		LinkCounts: pagecontents.LinkCounts{
			LocalLinks:    document.LocalLinks,
			ExternalLinks: document.ExternalLinks,
		},
	}
}
