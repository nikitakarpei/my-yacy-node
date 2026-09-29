package upstreamrequest

import "context"

type Observer interface {
	Failed(ctx context.Context, address string, step Step, cause error)
}

type Observers []Observer

func (observers Observers) Failed(ctx context.Context, address string, step Step, cause error) {
	for _, observer := range observers {
		observer.Failed(ctx, address, step, cause)
	}
}
