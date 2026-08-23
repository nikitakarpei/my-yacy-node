// Package documentextraction turns a fetched body into the document it holds.
package documentextraction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"strings"
)

const msgMediaTypeUnparsed = "content type unparsed, falling back to its leading segment"

var ErrUnsupportedMediaType = errors.New("unsupported media type")

func DocumentFrom(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (Document, error) {
	media := mediaType(ctx, contentType)
	extractor, extractable := documentExtractorFor(media)
	if !extractable {
		return Document{}, ErrUnsupportedMediaType
	}
	document, err := extractor.Extract(ctx, pageURL, contentType, body)
	if err != nil {
		return Document{}, fmt.Errorf("extract %s: %w", media, err)
	}
	return document, nil
}

func mediaType(ctx context.Context, contentType string) string {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		slog.WarnContext(ctx, msgMediaTypeUnparsed,
			slog.String("contentType", contentType),
			slog.Any("error", err),
		)
		return strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	return media
}
