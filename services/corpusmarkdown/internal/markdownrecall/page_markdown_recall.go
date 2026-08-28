// Package markdownrecall yields the markdown the corpus holds for a requested URL. The
// corpus stores a page under the URL the origin settled on, so a request for a URL that
// redirected finds nothing under its own name; this package follows the redirection the
// scrape recorded and recalls the page under the URL that holds it.
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

type PageRedirections interface {
	RedirectionOf(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
	) (canonicalurl.CanonicalURL, bool, error)
}

type RecalledPage struct {
	MarkdownURL canonicalurl.CanonicalURL
	StoredMarkdown
}

type PageMarkdownRecall struct {
	corpus       PageMarkdownCorpus
	redirections PageRedirections
}

func NewPageMarkdownRecall(
	corpus PageMarkdownCorpus,
	redirections PageRedirections,
) *PageMarkdownRecall {
	return &PageMarkdownRecall{corpus: corpus, redirections: redirections}
}

func (r *PageMarkdownRecall) PageOf(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) (RecalledPage, bool, error) {
	page, held, err := r.pageUnder(ctx, requestedURL)
	if err != nil || held {
		return page, held, err
	}
	markdownURL, redirected, err := r.redirections.RedirectionOf(ctx, requestedURL)
	if err != nil || !redirected {
		return RecalledPage{}, false, err
	}
	return r.pageUnder(ctx, markdownURL)
}

func (r *PageMarkdownRecall) pageUnder(
	ctx context.Context,
	markdownURL canonicalurl.CanonicalURL,
) (RecalledPage, bool, error) {
	stored, held, err := r.corpus.MarkdownOf(ctx, markdownURL)
	if err != nil || !held {
		return RecalledPage{}, false, err
	}
	return RecalledPage{MarkdownURL: markdownURL, StoredMarkdown: stored}, true, nil
}
