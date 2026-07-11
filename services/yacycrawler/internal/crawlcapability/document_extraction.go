package crawlcapability

import "context"

type DocumentExtraction interface {
	Extract(
		ctx context.Context,
		resourceURL, contentType string,
		body []byte,
	) ([]ExtractedDocument, error)
}
