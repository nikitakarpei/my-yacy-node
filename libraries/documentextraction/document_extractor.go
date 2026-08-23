package documentextraction

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type documentExtractor interface {
	DocumentFrom(
		ctx context.Context,
		body []byte,
		contentType string,
		pageURL canonicalurl.CanonicalURL,
	) (Document, error)
	MediaTypes() []string
	EmittedFormat() Format
}

func documentExtractorCatalog() []documentExtractor {
	return []documentExtractor{newHTMLExtraction()}
}

func documentExtractorFor(media string) (documentExtractor, bool) {
	for _, extractor := range documentExtractorCatalog() {
		for _, mediaType := range extractor.MediaTypes() {
			if mediaType == media {
				return extractor, true
			}
		}
	}
	return nil, false
}

func EmittedFormats() []Format {
	catalog := documentExtractorCatalog()
	formats := make([]Format, 0, len(catalog))
	for _, extractor := range catalog {
		formats = append(formats, extractor.EmittedFormat())
	}
	return formats
}
