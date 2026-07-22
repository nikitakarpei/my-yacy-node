package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefulltext"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagereadablehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagereadabletext"
)

func pageDerivationCatalog() []crawlcapability.PageDerivation {
	return []crawlcapability.PageDerivation{
		pagefulltext.NewDocumentHTMLDerivation(),
		pagereadablehtml.NewDocumentHTMLDerivation(),
		pagereadabletext.NewReadableHTMLDerivation(),
		pagereadabletext.NewFullTextDerivation(),
		pagemarkdown.NewReadableHTMLDerivation(),
	}
}
