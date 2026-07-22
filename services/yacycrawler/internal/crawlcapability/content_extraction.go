package crawlcapability

import (
	"context"
	"errors"
)

var (
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrUnextractable        = errors.New("unextractable")
)

type ContentExtraction interface {
	Extract(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) (ExtractedContent, error)
}
