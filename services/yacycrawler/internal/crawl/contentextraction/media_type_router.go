package contentextraction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"strings"
)

const msgMemberUnextracted = "container member dropped: extraction failed"

type MediaTypeRouter struct {
	extractors   map[string]Extractor
	containers   map[string]Container
	maxDepth     int
	maxDocuments int
}

func New(maxDepth, maxDocuments int) *MediaTypeRouter {
	return &MediaTypeRouter{
		extractors:   map[string]Extractor{},
		containers:   map[string]Container{},
		maxDepth:     maxDepth,
		maxDocuments: maxDocuments,
	}
}

func (r *MediaTypeRouter) RegisterExtractor(
	mediaType string,
	extractor Extractor,
) {
	r.extractors[mediaType] = extractor
}

func (r *MediaTypeRouter) RegisterContainer(
	mediaType string,
	container Container,
) {
	r.containers[mediaType] = container
}

func (r *MediaTypeRouter) RegisteredMediaTypes() int {
	return len(r.extractors) + len(r.containers)
}

func (r *MediaTypeRouter) Extract(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) ([]ExtractedDocument, error) {
	documents, err := r.route(ctx, 0, pageURL, contentType, body)
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (r *MediaTypeRouter) route(
	ctx context.Context,
	depth int,
	resourceURL, contentType string,
	body []byte,
) ([]ExtractedDocument, error) {
	media := mediaType(contentType)

	if extractor, ok := r.extractors[media]; ok {
		content, err := extractor.Extract(ctx, resourceURL, contentType, body)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", media, err)
		}
		return []ExtractedDocument{{
			URL:              resourceURL,
			ExtractedContent: content,
		}}, nil
	}

	container, ok := r.containers[media]
	if !ok {
		return nil, ErrUnsupportedMediaType
	}
	if depth >= r.maxDepth {
		return nil, ErrContainerOverflow
	}

	members, err := container.Expand(ctx, resourceURL, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("expand %s: %w", media, err)
	}

	var documents []ExtractedDocument
	for _, member := range members {
		fromMember, err := r.route(ctx, depth+1, member.URL, member.ContentType, member.Body)
		if err != nil {
			if errors.Is(err, ErrContainerOverflow) {
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
		if len(documents)+len(fromMember) > r.maxDocuments {
			return nil, ErrContainerOverflow
		}
		documents = append(documents, fromMember...)
	}
	return documents, nil
}

func mediaType(contentType string) string {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	return media
}
