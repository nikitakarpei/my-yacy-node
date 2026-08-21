package pagescrape

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/mediaextractors/html"
)

type RegisteredMediaExtractor interface {
	contentextraction.MediaExtractor
	MediaTypes() []string
	EmittedFormat() contentformatgraph.Format
}

func MediaExtractorCatalog() []RegisteredMediaExtractor {
	return []RegisteredMediaExtractor{
		html.New(),
	}
}

func mediaExtractorsByMediaType() map[string]contentextraction.MediaExtractor {
	byMediaType := map[string]contentextraction.MediaExtractor{}
	for _, extractor := range MediaExtractorCatalog() {
		for _, mediaType := range extractor.MediaTypes() {
			byMediaType[mediaType] = extractor
		}
	}
	return byMediaType
}

func emittedFormats() []contentformatgraph.Format {
	catalog := MediaExtractorCatalog()
	formats := make([]contentformatgraph.Format, 0, len(catalog))
	for _, extractor := range catalog {
		formats = append(formats, extractor.EmittedFormat())
	}
	return formats
}
