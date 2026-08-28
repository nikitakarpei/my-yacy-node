//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	searchWebToolName = "search_web"
	readPageToolName  = "read_page"

	pageFetched    = "page-fetched"
	fetchNotNeeded = "fetch-not-needed"
)

type searchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type searchAnswer struct {
	Results []searchResult `json:"results"`
}

type searchCall struct {
	Query       string `json:"query"`
	ResultLimit int    `json:"resultLimit,omitempty"`
}

type pageCall struct {
	URL            string `json:"url"`
	CharacterLimit int    `json:"characterLimit,omitempty"`
	Version        string `json:"version,omitempty"`
}

type pageAnswer struct {
	URL                    string    `json:"url"`
	Version                string    `json:"version"`
	StoredAt               time.Time `json:"storedAt"`
	FetchOutcome           string    `json:"fetchOutcome"`
	Markdown               string    `json:"markdown"`
	MarkdownCharacterCount int       `json:"markdownCharacterCount"`
	Truncated              bool      `json:"truncated"`
}

func openToolSession(t *testing.T, ctx context.Context, endpointURL string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "webresearchmcp-e2e", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpointURL}, nil)
	if err != nil {
		t.Fatalf("connect to the tools at %s: %v", endpointURL, err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func toolNamesOf(t *testing.T, ctx context.Context, session *mcp.ClientSession) []string {
	t.Helper()
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func searchResultsFor(
	t *testing.T,
	ctx context.Context,
	session *mcp.ClientSession,
	call searchCall,
) []searchResult {
	t.Helper()
	var answer searchAnswer
	callTool(t, ctx, session, searchWebToolName, call, &answer)
	return answer.Results
}

func pageAnswerFor(
	t *testing.T,
	ctx context.Context,
	session *mcp.ClientSession,
	call pageCall,
) pageAnswer {
	t.Helper()
	var answer pageAnswer
	callTool(t, ctx, session, readPageToolName, call, &answer)
	return answer
}

func callTool(
	t *testing.T,
	ctx context.Context,
	session *mcp.ClientSession,
	name string,
	arguments any,
	answer any,
) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("call the %s tool: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("the %s tool answered with an error: %v", name, result.Content)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal the answer of the %s tool: %v", name, err)
	}
	if err := json.Unmarshal(structured, answer); err != nil {
		t.Fatalf("decode the answer of the %s tool: %v", name, err)
	}
}
