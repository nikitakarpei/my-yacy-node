// Package corpustext reads the search endpoint of the operator's corpustext service, which
// matches the full text of the pages the operator's own stack scraped.
package corpustext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const searchResponseByteLimit = 8 << 20

type CorpusText struct {
	corpusTextURL string
	client        *http.Client
}

func NewCorpusText(corpusTextURL string, searchDeadline time.Duration) *CorpusText {
	return &CorpusText{
		corpusTextURL: corpusTextURL,
		client:        &http.Client{Timeout: searchDeadline},
	}
}

func (e *CorpusText) SearchResultsFor(
	ctx context.Context,
	query string,
) ([]websearch.SearchResult, error) {
	request, err := e.searchRequestFor(ctx, query)
	if err != nil {
		return nil, err
	}
	response, err := e.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ask corpustext for %q: %w", query, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("corpustext answered %s for %q", response.Status, query)
	}
	return searchResultsIn(response)
}

func (e *CorpusText) searchRequestFor(
	ctx context.Context,
	query string,
) (*http.Request, error) {
	searchURL := e.corpusTextURL + searchdocument.SearchPath + "?" + url.Values{
		searchdocument.SearchQueryParam: {query},
	}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build the corpustext request for %q: %w", query, err)
	}
	return request, nil
}

func searchResultsIn(response *http.Response) ([]websearch.SearchResult, error) {
	var answer searchdocument.PageMatches
	body := io.LimitReader(response.Body, searchResponseByteLimit)
	if err := json.NewDecoder(body).Decode(&answer); err != nil {
		return nil, fmt.Errorf("read the corpustext answer: %w", err)
	}
	results := make([]websearch.SearchResult, 0, len(answer.Matches))
	for _, match := range answer.Matches {
		results = append(results, websearch.SearchResult{
			URL:     match.URL,
			Title:   match.Title,
			Snippet: match.Snippet,
		})
	}
	return results, nil
}
