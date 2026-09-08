// Package http reads one page's markdown from the corpus that holds it, over the corpus
// contract. A page the corpus does not hold is an answer and not a failure, so this package
// turns the contract's not-found answer into that answer.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

type MarkdownCorpus struct {
	recallAddress string
	recalls       *http.Client
}

func NewMarkdownCorpus(corpusAddress string, recallDeadline time.Duration) *MarkdownCorpus {
	return &MarkdownCorpus{
		recallAddress: "http://" + corpusAddress + pagemarkdownstore.RecalledPagePath,
		recalls:       &http.Client{Timeout: recallDeadline},
	}
}

func (c *MarkdownCorpus) PageMarkdownAt(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) (pageread.PageMarkdown, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.recallAddressOf(pageURL), nil,
	)
	if err != nil {
		return pageread.PageMarkdown{}, fmt.Errorf("ask the corpus for %q: %w", pageURL, err)
	}
	response, err := c.recalls.Do(request)
	if err != nil {
		return pageread.PageMarkdown{}, fmt.Errorf("recall %q: %w", pageURL, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusNotFound {
		return pageread.PageMarkdown{}, pageread.ErrPageNotInCorpus
	}
	if response.StatusCode != http.StatusOK {
		return pageread.PageMarkdown{}, fmt.Errorf("recall %q: %s", pageURL, response.Status)
	}
	var recalled pagemarkdownstore.RecalledPage
	if err := json.NewDecoder(response.Body).Decode(&recalled); err != nil {
		return pageread.PageMarkdown{}, fmt.Errorf(
			"read the corpus answer for %q: %w",
			pageURL,
			err,
		)
	}
	return pageMarkdownFrom(recalled), nil
}

func (c *MarkdownCorpus) recallAddressOf(pageURL canonicalurl.CanonicalURL) string {
	query := url.Values{pagemarkdownstore.RequestedURLQueryParam: {pageURL.String()}}
	return c.recallAddress + "?" + query.Encode()
}

func pageMarkdownFrom(recalled pagemarkdownstore.RecalledPage) pageread.PageMarkdown {
	return pageread.PageMarkdown{
		Markdown: recalled.Markdown,
		Version:  recalled.Version,
		StoredAt: recalled.StoredAt,
	}
}
