package tlsclient

import "context"

type Observer interface {
	PageFetched(ctx context.Context, address string, status int)
	FetchFailed(ctx context.Context, address string, cause error)
	FetchCancelled(ctx context.Context, address string)
}

type Observers []Observer

func (observers Observers) PageFetched(ctx context.Context, address string, status int) {
	for _, observer := range observers {
		observer.PageFetched(ctx, address, status)
	}
}

func (observers Observers) FetchFailed(ctx context.Context, address string, cause error) {
	for _, observer := range observers {
		observer.FetchFailed(ctx, address, cause)
	}
}

func (observers Observers) FetchCancelled(ctx context.Context, address string) {
	for _, observer := range observers {
		observer.FetchCancelled(ctx, address)
	}
}
