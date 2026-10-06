package markdown

import (
	"context"
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/bodies/text"
)

type DocumentHTMLDerivation struct{}

func FromDocumentHTML() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatMarkdown
}

func (DocumentHTMLDerivation) BodyFrom(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	source documentextraction.Body,
) (documentextraction.Body, bool, error) {
	markdown, err := htmltomarkdown.ConvertNode(source.(htmltree.Body).CopiedTree())
	if err != nil {
		return nil, false, fmt.Errorf("convert html to markdown: %w", err)
	}
	return text.New(markdown), true, nil
}
