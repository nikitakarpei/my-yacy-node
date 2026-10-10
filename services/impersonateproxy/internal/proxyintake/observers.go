package proxyintake

import "context"

type Observer interface {
	MethodRefused(ctx context.Context, method string)
	TargetRefused(ctx context.Context, target string)
	ResponseLeftIncomplete(
		ctx context.Context,
		address string,
		incompleteResponseCause IncompleteResponseCause,
		cause error,
	)
}

type Observers []Observer

func (observers Observers) MethodRefused(ctx context.Context, method string) {
	for _, observer := range observers {
		observer.MethodRefused(ctx, method)
	}
}

func (observers Observers) TargetRefused(ctx context.Context, target string) {
	for _, observer := range observers {
		observer.TargetRefused(ctx, target)
	}
}

func (observers Observers) ResponseLeftIncomplete(
	ctx context.Context,
	address string,
	incompleteResponseCause IncompleteResponseCause,
	cause error,
) {
	for _, observer := range observers {
		observer.ResponseLeftIncomplete(ctx, address, incompleteResponseCause, cause)
	}
}
