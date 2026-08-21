package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/mediaextractors/html"
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
