package pagehtml

import "context"

type MediaTypeObserver interface {
	MediaTypeUnparsed(ctx context.Context, contentType string, cause error)
}

type MediaTypeObservers []MediaTypeObserver

func (observers MediaTypeObservers) MediaTypeUnparsed(
	ctx context.Context,
	contentType string,
	cause error,
) {
	for _, observer := range observers {
		observer.MediaTypeUnparsed(ctx, contentType, cause)
	}
}
