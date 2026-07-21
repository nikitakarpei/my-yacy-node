package markdownrecall

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

type MarkdownPage struct {
	CanonicalURL string
	Markdown     string
}

type MarkdownObjects interface {
	GetBytes(ctx context.Context, name string, opts ...jetstream.GetObjectOpt) ([]byte, error)
}

type Source struct {
	objects  MarkdownObjects
	maxBytes int64
}

func NewSource(objects MarkdownObjects, maxBytes int64) *Source {
	return &Source{objects: objects, maxBytes: maxBytes}
}

func (s *Source) Fetch(
	ctx context.Context,
	targetURL string,
) (pagerecall.Representation, bool, error) {
	markdown, err := s.objects.GetBytes(ctx, pagemarkdownstore.ObjectName(targetURL))
	if errors.Is(err, jetstream.ErrObjectNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get markdown for %q: %w", targetURL, err)
	}
	if int64(len(markdown)) > s.maxBytes {
		return nil, false, fmt.Errorf(
			"markdown for %q is %d bytes, exceeding limit %d",
			targetURL, len(markdown), s.maxBytes,
		)
	}
	return MarkdownPage{CanonicalURL: targetURL, Markdown: string(markdown)}, true, nil
}
