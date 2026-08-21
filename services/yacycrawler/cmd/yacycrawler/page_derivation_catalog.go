package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagederivations/fulltext"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagederivations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagederivations/readablehtml"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagederivations/readabletext"
)

func pageDerivationCatalog() []contentformatgraph.Derivation {
	return []contentformatgraph.Derivation{
		fulltext.NewDocumentHTMLDerivation(),
		readablehtml.NewDocumentHTMLDerivation(),
		readabletext.NewReadableHTMLDerivation(),
		readabletext.NewFullTextDerivation(),
		markdown.NewReadableHTMLDerivation(),
		markdown.NewDocumentHTMLDerivation(),
	}
}
