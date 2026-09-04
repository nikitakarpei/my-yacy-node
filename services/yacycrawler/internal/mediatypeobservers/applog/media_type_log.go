package applog

import (
	"context"
	"log/slog"
)

const msgMediaTypeUnparsed = "content type unparsed, falling back to its leading segment"

type MediaTypeLog struct{}

func (MediaTypeLog) MediaTypeUnparsed(
	ctx context.Context,
	contentType string,
	cause error,
) {
	slog.WarnContext(ctx, msgMediaTypeUnparsed,
		slog.String("contentType", contentType),
		slog.Any("error", cause),
	)
}
