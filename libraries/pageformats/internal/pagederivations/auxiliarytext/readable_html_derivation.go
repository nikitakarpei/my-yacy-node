package auxiliarytext

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articleauxiliary"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlefootnotes"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/bodies/text"
)

type ReadableHTMLDerivation struct{}

func FromReadableHTML() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatAuxiliaryText
}

func (ReadableHTMLDerivation) BodyFrom(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	source documentextraction.Body,
) (documentextraction.Body, bool, error) {
	article := source.(htmltree.Body).CopiedTree()
	auxiliaryParts := articleauxiliary.PartsOf(article)
	articlefootnotes.RemoveFootnoteMarks(article)
	return text.New([]byte(auxiliaryParts.Text())), true, nil
}
