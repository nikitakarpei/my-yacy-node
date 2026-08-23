// Package pageformats derives the content formats a page's document can be read in.
package pageformats

import (
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/fulltext"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readablehtml"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readabletext"
)

func New() (DerivableFormats, error) {
	derivableFormats := DerivableFormatsOf(pageDerivationCatalog())
	if err := derivableFormats.EnsureNoDanglingFormat(
		documentextraction.EmittedFormats(),
		derivableFormats.TargetFormats(),
	); err != nil {
		return DerivableFormats{}, err
	}
	return derivableFormats, nil
}

func pageDerivationCatalog() []FormatDerivation {
	return []FormatDerivation{
		fulltext.FromDocumentHTML(),
		readablehtml.FromDocumentHTML(),
		readabletext.FromReadableHTML(),
		readabletext.FromFullText(),
		markdown.FromReadableHTML(),
		markdown.FromDocumentHTML(),
	}
}
