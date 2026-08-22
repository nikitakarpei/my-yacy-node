// Package contentextraction turns a fetched body into the document it holds.
package contentextraction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"strings"
)

const msgMediaTypeUnparsed = "content type unparsed, falling back to its leading segment"

var ErrNoExtractableMediaType = errors.New("no media type is extractable")

type DocumentExtractor struct {
	extractors map[string]MediaExtractor
}

func New(extractors map[string]MediaExtractor) (*DocumentExtractor, error) {
	if len(extractors) == 0 {
		return nil, ErrNoExtractableMediaType
	}
	return &DocumentExtractor{extractors: extractors}, nil
}

func (e *DocumentExtractor) DocumentFrom(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (ExtractedDocument, error) {
	media := mediaType(ctx, contentType)
	extractor, extractable := e.extractors[media]
	if !extractable {
		return ExtractedDocument{}, ErrUnsupportedMediaType
	}
	document, err := extractor.Extract(ctx, pageURL, contentType, body)
	if err != nil {
		return ExtractedDocument{}, fmt.Errorf("extract %s: %w", media, err)
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
