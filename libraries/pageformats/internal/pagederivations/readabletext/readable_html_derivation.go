package readabletext

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articleauxiliary"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlefootnotes"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlemediawikichrome"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlepermalinks"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/bodies/text"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/htmlflattening"
)

type ReadableHTMLDerivation struct{}

func FromReadableHTML() ReadableHTMLDerivation {
	return ReadableHTMLDerivation{}
}

func (ReadableHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatReadableHTML
}

func (ReadableHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatReadableText
}

func (ReadableHTMLDerivation) BodyFrom(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	source documentextraction.Body,
) (documentextraction.Body, bool, error) {
	article := source.(htmltree.Body).CopiedTree()
	auxiliaryParts := articleauxiliary.PartsOf(article)
	articlefootnotes.RemoveFootnoteMarks(article)
	auxiliaryParts.Remove()
	articlepermalinks.RemovePermalinks(article)
	articlemediawikichrome.RemoveMediaWikiChrome(article)
	readableText := htmlflattening.Flatten(article)
	return text.New([]byte(readableText)), true, nil
}
