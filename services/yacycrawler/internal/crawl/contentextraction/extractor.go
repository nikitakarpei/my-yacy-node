package contentextraction

import (
	"context"
	"errors"
)

var ErrUnsupportedMediaType = errors.New("unsupported media type")

type Extractor interface {
	Extract(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) (ExtractedContent, error)
}
