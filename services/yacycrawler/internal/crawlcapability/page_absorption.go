package crawlcapability

import "context"

type PageAbsorption interface {
	Absorb(ctx context.Context, outcome FetchOutcome) (links []string, err error)
}
