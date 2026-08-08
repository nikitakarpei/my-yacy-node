// Package jetstream yields the markdown page a JetStream object store holds for a URL.
package jetstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

type Corpus struct {
	objects  jetstream.ObjectStore
	maxBytes int64
}

func NewCorpus(objects jetstream.ObjectStore, maxBytes int64) *Corpus {
	return &Corpus{objects: objects, maxBytes: maxBytes}
}

func (c *Corpus) RepresentationKind() recall.RepresentationKind {
	return markdown.Kind
}

func (c *Corpus) RepresentationOf(
	ctx context.Context,
	resolvedURL string,
) (recall.Representation, bool, error) {
	pageMarkdown, err := c.objects.GetBytes(ctx, pagemarkdownstore.ObjectName(resolvedURL))
	if errors.Is(err, jetstream.ErrObjectNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get markdown for %q: %w", resolvedURL, err)
	}
	if int64(len(pageMarkdown)) > c.maxBytes {
		return nil, false, fmt.Errorf(
			"markdown for %q is %d bytes, exceeding limit %d",
			resolvedURL, len(pageMarkdown), c.maxBytes,
		)
	}
	return markdown.Page{CanonicalURL: resolvedURL, Markdown: string(pageMarkdown)}, true, nil
}
