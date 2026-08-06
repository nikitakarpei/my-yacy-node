package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediaextractors/html"
)

type registeredMediaExtractor interface {
	contentextraction.MediaExtractor
	MediaTypes() []string
	EmittedFormat() contentformatgraph.Format
}

func mediaExtractorCatalog() []registeredMediaExtractor {
	return []registeredMediaExtractor{
		html.New(),
	}
}
