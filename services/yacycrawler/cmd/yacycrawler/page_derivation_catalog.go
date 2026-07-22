package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/fulltext"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/readablehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagederivations/readabletext"
)

func pageDerivationCatalog() []contentformatgraph.Derivation {
	return []contentformatgraph.Derivation{
		fulltext.NewDocumentHTMLDerivation(),
		readablehtml.NewDocumentHTMLDerivation(),
		readabletext.NewReadableHTMLDerivation(),
		readabletext.NewFullTextDerivation(),
		markdown.NewReadableHTMLDerivation(),
	}
}
