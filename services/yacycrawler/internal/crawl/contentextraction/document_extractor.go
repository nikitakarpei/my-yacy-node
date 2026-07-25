// Package contentextraction turns a fetched body into documents, expanding containers on the way.
package contentextraction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"strings"
)

const (
	msgMemberUnextracted = "container member dropped: extraction failed"
	msgMediaTypeUnparsed = "content type unparsed, falling back to its leading segment"
)

var (
	ErrNoExtractableMediaType  = errors.New("no media type is extractable")
	ErrDocumentBudgetExhausted = errors.New("document budget exhausted")
)

type DocumentExtractor struct {
	extractors   map[string]MediaExtractor
	containers   map[string]ContainerExpander
	maxDepth     int
	maxDocuments int
}

func New(
	extractors map[string]MediaExtractor,
	containers map[string]ContainerExpander,
	maxDepth, maxDocuments int,
) (*DocumentExtractor, error) {
	if len(extractors)+len(containers) == 0 {
		return nil, ErrNoExtractableMediaType
	}
	return &DocumentExtractor{
		extractors:   extractors,
		containers:   containers,
		maxDepth:     maxDepth,
		maxDocuments: maxDocuments,
	}, nil
}

func (e *DocumentExtractor) ExtractDocuments(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) ([]ExtractedDocument, error) {
	run := extraction{extractor: e, remainingDocuments: e.maxDocuments}
	return run.extractDocuments(ctx, 0, pageURL, contentType, body)
}

type extraction struct {
	extractor          *DocumentExtractor
	remainingDocuments int
}

func (r *extraction) extractDocuments(
	ctx context.Context,
	depth int,
	resourceURL, contentType string,
	body []byte,
) ([]ExtractedDocument, error) {
	media := mediaType(ctx, contentType)

	if extractor, ok := r.extractor.extractors[media]; ok {
		if r.remainingDocuments == 0 {
			return nil, ErrDocumentBudgetExhausted
		}
		content, err := extractor.Extract(ctx, resourceURL, contentType, body)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", media, err)
		}
		r.remainingDocuments--
		return []ExtractedDocument{{
			URL:              resourceURL,
			ExtractedContent: content,
		}}, nil
	}

	container, ok := r.extractor.containers[media]
	if !ok {
		return nil, ErrUnsupportedMediaType
	}
	if depth >= r.extractor.maxDepth {
		return nil, ErrNestingTooDeep
	}

	members, err := container.Expand(ctx, resourceURL, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("expand %s: %w", media, err)
	}

	var documents []ExtractedDocument
	for _, member := range members {
		fromMember, err := r.extractDocuments(
			ctx, depth+1, member.URL, member.ContentType, member.Body,
		)
		if err != nil {
			if errors.Is(err, ErrNestingTooDeep) || errors.Is(err, ErrDocumentBudgetExhausted) {
				return nil, err
			}
			if !errors.Is(err, ErrUnsupportedMediaType) {
				slog.WarnContext(ctx, msgMemberUnextracted,
					slog.String("member", member.URL),
					slog.Any("error", err),
				)
			}
			continue
		}
		documents = append(documents, fromMember...)
	}
	return documents, nil
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
