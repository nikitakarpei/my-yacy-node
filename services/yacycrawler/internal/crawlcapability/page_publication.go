package crawlcapability

import "context"

type PagePublication[R any] interface {
	Publish(ctx context.Context, representation R) error
}
