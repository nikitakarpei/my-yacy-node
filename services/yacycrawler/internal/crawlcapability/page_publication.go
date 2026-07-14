package crawlcapability

import "context"

type PagePublication interface {
	Name() string
	Accepts(format PageContentFormat) bool
	Publish(ctx context.Context, page ExtractedPage) error
}
