package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefulltext"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagereadable"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func pageRenderingCatalog() []crawlcapability.PageRendering {
	return []crawlcapability.PageRendering{
		pagefulltext.NewHTMLRendering(),
		pagereadable.NewHTMLRendering(),
		pagetext.NewReadableTextRendering(),
		pagetext.NewFullTextFallbackRendering(),
		pagemarkdown.NewHTMLRendering(),
	}
}
