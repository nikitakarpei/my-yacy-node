package judgedqueries_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
)

type pageExtraction struct {
	formatDerivations pageformats.FormatDerivationCatalog
}

type extractedPage struct {
	title      string
	text       string
	linkCounts pagecontents.LinkCounts
}

func pageExtractionOfTheFormats(t *testing.T) pageExtraction {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page format derivations: %v", err)
	}

	return pageExtraction{formatDerivations: formatDerivations}
}

func (e pageExtraction) extractedPageOf(ctx context.Context, page storedPage) extractedPage {
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
		text:  e.textOfTheDocument(ctx, document, pageURL),
		linkCounts: pagecontents.LinkCounts{
			LocalLinks:    document.LocalLinks,
			ExternalLinks: document.ExternalLinks,
		},
	}
}

func (e pageExtraction) textOfTheDocument(
	ctx context.Context,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) string {
	readableText, readableTextDerived := e.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, document, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		return string(readableText)
	}
	fullText, fullTextDerived := e.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, document, pageURL,
	)
	if !fullTextDerived {
		return ""
	}

	return string(bytes.TrimSpace(fullText))
}
