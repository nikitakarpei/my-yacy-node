package contentextraction

import (
	"context"
	"errors"
)

var ErrUnsupportedMediaType = errors.New("unsupported media type")

type MediaExtractor interface {
	Extract(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) (ExtractedDocument, error)
}
