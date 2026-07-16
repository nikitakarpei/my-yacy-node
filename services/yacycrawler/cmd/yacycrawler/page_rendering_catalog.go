package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func pageRenderingCatalog() []crawlcapability.PageRendering {
	return []crawlcapability.PageRendering{
		pagetext.NewHTMLRendering(),
		pagetext.NewTextRendering(),
		pagemarkdown.NewHTMLRendering(),
	}
}
