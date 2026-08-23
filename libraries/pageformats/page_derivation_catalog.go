// Package pageformats derives the content formats a page's document can be read in.
package pageformats

import (
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/fulltext"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readablehtml"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/pagederivations/readabletext"
)

func New() (FormatDerivations, error) {
	derivations := FormatDerivationsOf(pageDerivationCatalog())
	if err := derivations.EnsureNoDanglingFormat(
		documentextraction.EmittedFormats(),
		derivations.TargetFormats(),
	); err != nil {
		return FormatDerivations{}, err
	}
	return derivations, nil
}

func pageDerivationCatalog() []Derivation {
	return []Derivation{
		fulltext.NewDocumentHTMLDerivation(),
		readablehtml.NewDocumentHTMLDerivation(),
		readabletext.NewReadableHTMLDerivation(),
		readabletext.NewFullTextDerivation(),
		markdown.NewReadableHTMLDerivation(),
		markdown.NewDocumentHTMLDerivation(),
	}
}
