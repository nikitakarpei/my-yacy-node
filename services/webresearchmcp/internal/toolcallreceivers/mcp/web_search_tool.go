package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

type searchWebArguments struct {
	Query       string `json:"query"`
	ResultLimit int    `json:"resultLimit,omitempty"`
}

type searchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type searchAnswer struct {
	Results []searchResult `json:"results"`
}

type webSearchTool struct {
	search    WebSearch
	admission *toolCallAdmission
}

func (t *webSearchTool) searchWeb(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	arguments searchWebArguments,
) (*mcp.CallToolResult, searchAnswer, error) {
	if err := t.admission.admit(ctx); err != nil {
		return nil, searchAnswer{}, err
	}
	defer t.admission.release()

	searchResults, err := t.search.SearchResultsFor(ctx, arguments.Query, arguments.ResultLimit)
	if err != nil {
		return nil, searchAnswer{}, err
	}
	return nil, searchAnswerFrom(searchResults), nil
}

func searchAnswerFrom(searchResults []websearch.SearchResult) searchAnswer {
	results := make([]searchResult, 0, len(searchResults))
	for _, result := range searchResults {
		results = append(results, searchResult{
			URL:     result.URL,
			Title:   result.Title,
			Snippet: result.Snippet,
		})
	}
	return searchAnswer{Results: results}
}
