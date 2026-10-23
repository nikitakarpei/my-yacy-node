// Package hitsreply sends a search to an index that answers with the hits reply that
// Elasticsearch and Manticore share, and turns each hit into a page match.
package hitsreply

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

const detailLimit = 4096

func PageMatchesFor(
	ctx context.Context,
	client *http.Client,
	target string,
	search []byte,
) ([]searchdocument.PageMatch, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(search))
	if err != nil {
		return nil, fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusMultipleChoices {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, detailLimit))
		return nil, fmt.Errorf("search: status %d: %s", resp.StatusCode, detail)
	}
	var reply hitsReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return nil, fmt.Errorf("decode search reply: %w", err)
	}
	return reply.pageMatches(), nil
}

type hitsReply struct {
	Hits struct {
		Hits []hit `json:"hits"`
	} `json:"hits"`
}

type hit struct {
	Source struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"_source"`
	Highlight struct {
		Content []string `json:"content"`
	} `json:"highlight"`
}

func (reply hitsReply) pageMatches() []searchdocument.PageMatch {
	matches := make([]searchdocument.PageMatch, 0, len(reply.Hits.Hits))
	for _, hit := range reply.Hits.Hits {
		matches = append(matches, searchdocument.PageMatch{
			URL:     hit.Source.URL,
			Title:   hit.Source.Title,
			Snippet: hit.snippet(),
		})
	}
	return matches
}

func (hit hit) snippet() string {
	if len(hit.Highlight.Content) == 0 {
		return ""
	}
	return hit.Highlight.Content[0]
}
