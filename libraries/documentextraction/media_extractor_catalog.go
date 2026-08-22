package documentextraction

type registeredMediaExtractor interface {
	MediaExtractor
	MediaTypes() []string
	EmittedFormat() Format
}

func mediaExtractorCatalog() []registeredMediaExtractor {
	return []registeredMediaExtractor{
		newHTMLExtraction(),
	}
}

func mediaExtractorsByMediaType() map[string]MediaExtractor {
	byMediaType := map[string]MediaExtractor{}
	for _, extractor := range mediaExtractorCatalog() {
		for _, mediaType := range extractor.MediaTypes() {
			byMediaType[mediaType] = extractor
		}
	}
	return byMediaType
}

func EmittedFormats() []Format {
	catalog := mediaExtractorCatalog()
	formats := make([]Format, 0, len(catalog))
	for _, extractor := range catalog {
		formats = append(formats, extractor.EmittedFormat())
	}
	return formats
}
