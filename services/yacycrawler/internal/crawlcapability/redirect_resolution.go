package crawlcapability

import "context"

type RedirectResolution interface {
	Record(ctx context.Context, requested, canonical string) error
}
