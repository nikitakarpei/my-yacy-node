package failurereporting

import "context"

type Observer interface {
	RoundTripFailed(ctx context.Context, address string, step Step, cause error)
}

type Observers []Observer

func (observers Observers) RoundTripFailed(
	ctx context.Context,
	address string,
	step Step,
	cause error,
) {
	for _, observer := range observers {
		observer.RoundTripFailed(ctx, address, step, cause)
	}
}
