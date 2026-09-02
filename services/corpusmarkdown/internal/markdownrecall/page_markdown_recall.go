// Package markdownrecall yields the markdown the corpus holds for a requested URL, under
// the URL the corpus stored it.
package markdownrecall

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type StoredMarkdown struct {
	Markdown []byte
	StoredAt time.Time
	Version  string
}

type PageMarkdownCorpus interface {
	MarkdownOf(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (StoredMarkdown, bool, error)
}

type RecalledPage struct {
	MarkdownURL canonicalurl.CanonicalURL
	StoredMarkdown
}

type PageMarkdownRecall struct {
	corpus PageMarkdownCorpus
}

func NewPageMarkdownRecall(corpus PageMarkdownCorpus) *PageMarkdownRecall {
	return &PageMarkdownRecall{corpus: corpus}
}

func (r *PageMarkdownRecall) PageOf(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) (RecalledPage, bool, error) {
	stored, held, err := r.corpus.MarkdownOf(ctx, requestedURL)
	if err != nil || !held {
		return RecalledPage{}, false, err
	}
	return RecalledPage{MarkdownURL: requestedURL, StoredMarkdown: stored}, true, nil
}
