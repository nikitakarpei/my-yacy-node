package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdownexcerpt"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	toolcallreceiversmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/toolcallreceivers/mcp"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const (
	pageAddress         = "https://example.org/page"
	resultURL           = "https://example.org/result"
	resultTitle         = "Research subject"
	resultSnippet       = "A page about the research subject."
	pageMarkdown        = "# Research"
	pageVersion         = "version-1"
	toolCallConcurrency = 2
	serviceVersion      = "test"
)

type searchAnswering struct {
	results     []websearch.SearchResult
	failure     error
	resultLimit int
}

func (s *searchAnswering) SearchResultsFor(
	_ context.Context,
	_ string,
	searchResultLimit int,
) ([]websearch.SearchResult, error) {
	s.resultLimit = searchResultLimit
	return s.results, s.failure
}

type pageAnswering struct {
	page    pageread.PageAnswer
	failure error
	call    pageread.PageCall
}

func (p *pageAnswering) PageAnswerFor(
	_ context.Context,
	call pageread.PageCall,
) (pageread.PageAnswer, error) {
	p.call = call
	return p.page, p.failure
}

func openToolSession(
	t *testing.T,
	search toolcallreceiversmcp.WebSearch,
	pageReader toolcallreceiversmcp.PageReader,
) *sdkmcp.ClientSession {
	t.Helper()
	server := httptest.NewServer(toolcallreceiversmcp.NewToolCallMux(toolcallreceiversmcp.Config{
		Search:              search,
		PageReader:          pageReader,
		ServiceVersion:      serviceVersion,
		ToolCallConcurrency: toolCallConcurrency,
	}))
	t.Cleanup(server.Close)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	session, err := client.Connect(
		context.Background(),
		&sdkmcp.StreamableClientTransport{
			Endpoint: server.URL + toolcallreceiversmcp.ToolEndpointPath,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("connect to the tools: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func answerOf(
	t *testing.T,
	session *sdkmcp.ClientSession,
	name string,
	arguments any,
	answer any,
) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(
		context.Background(),
		&sdkmcp.CallToolParams{Name: name, Arguments: arguments},
	)
	if err != nil {
		t.Fatalf("call the %s tool: %v", name, err)
	}
	if result.IsError {
		return result
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal the answer of the %s tool: %v", name, err)
	}
	if err := json.Unmarshal(structured, answer); err != nil {
		t.Fatalf("decode the answer of the %s tool: %v", name, err)
	}
	return result
}

func pageAnswerUnderTest(t *testing.T) pageread.PageAnswer {
	t.Helper()
	pageURL, err := canonicalurl.CanonicalURLOf(pageAddress)
	if err != nil {
		t.Fatalf("read the page address: %v", err)
	}
	return pageread.PageAnswer{
		URL:          pageURL,
		Version:      pageVersion,
		StoredAt:     time.Now().UTC().Truncate(time.Second),
		FetchOutcome: pageread.PageFetched,
		Excerpt:      markdownexcerpt.MarkdownExcerptOf(pageMarkdown, len(pageMarkdown)),
	}
}

func TestBothResearchToolsAreServed(t *testing.T) {
	session := openToolSession(t, &searchAnswering{}, &pageAnswering{})

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	served := make(map[string]bool, len(listed.Tools))
	for _, tool := range listed.Tools {
		served[tool.Name] = true
	}
	for _, wanted := range []string{"search_web", "read_page"} {
		if !served[wanted] {
			t.Errorf("served tools = %v, want them to carry %q", served, wanted)
		}
	}
}

func TestSearchToolAnswersWithTheResultsOfTheSearch(t *testing.T) {
	search := &searchAnswering{results: []websearch.SearchResult{
		{URL: resultURL, Title: resultTitle, Snippet: resultSnippet},
	}}
	session := openToolSession(t, search, &pageAnswering{})

	var answer struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	answerOf(t, session, "search_web", map[string]any{
		"query":       "research subject",
		"resultLimit": 5,
	}, &answer)

	if len(answer.Results) != 1 {
		t.Fatalf("results = %v, want the one the search answers", answer.Results)
	}
	if answer.Results[0].URL != resultURL {
		t.Errorf("result url = %q, want %q", answer.Results[0].URL, resultURL)
	}
	if answer.Results[0].Title != resultTitle {
		t.Errorf("result title = %q, want %q", answer.Results[0].Title, resultTitle)
	}
	if answer.Results[0].Snippet != resultSnippet {
		t.Errorf("result snippet = %q, want %q", answer.Results[0].Snippet, resultSnippet)
	}
	if search.resultLimit != 5 {
		t.Errorf("result limit = %d, want the 5 the call names", search.resultLimit)
	}
}

func TestSearchThatFailsIsAnsweredAsAnError(t *testing.T) {
	search := &searchAnswering{failure: errors.New("engine away")}
	session := openToolSession(t, search, &pageAnswering{})

	var answer any
	result := answerOf(t, session, "search_web", map[string]any{"query": "q"}, &answer)
	if !result.IsError {
		t.Error("the tool answered without an error, want the failed search to be an error")
	}
}

func TestPageToolAnswersWithTheMarkdownAndWhatBecameOfTheFetch(t *testing.T) {
	page := pageAnswerUnderTest(t)
	pageReader := &pageAnswering{page: page}
	session := openToolSession(t, &searchAnswering{}, pageReader)

	var answer struct {
		URL                    string    `json:"url"`
		Version                string    `json:"version"`
		StoredAt               time.Time `json:"storedAt"`
		FetchOutcome           string    `json:"fetchOutcome"`
		Markdown               string    `json:"markdown"`
		MarkdownCharacterCount int       `json:"markdownCharacterCount"`
		Truncated              bool      `json:"truncated"`
	}
	answerOf(t, session, "read_page", map[string]any{
		"url":            pageAddress,
		"characterLimit": 40,
		"toleratedAge":   "30m",
	}, &answer)

	if answer.URL != pageAddress {
		t.Errorf("page url = %q, want %q", answer.URL, pageAddress)
	}
	if answer.Version != pageVersion {
		t.Errorf("version = %q, want %q", answer.Version, pageVersion)
	}
	if !answer.StoredAt.Equal(page.StoredAt) {
		t.Errorf("stored at = %v, want %v", answer.StoredAt, page.StoredAt)
	}
	if answer.FetchOutcome != string(pageread.PageFetched) {
		t.Errorf("fetch outcome = %q, want %q", answer.FetchOutcome, pageread.PageFetched)
	}
	if answer.Markdown != pageMarkdown {
		t.Errorf("markdown = %q, want %q", answer.Markdown, pageMarkdown)
	}
	if answer.MarkdownCharacterCount != len(pageMarkdown) {
		t.Errorf(
			"markdown character count = %d, want %d",
			answer.MarkdownCharacterCount, len(pageMarkdown),
		)
	}
	if answer.Truncated {
		t.Error("the answer says it is truncated, want the whole markdown")
	}
	if pageReader.call.CharacterLimit != 40 {
		t.Errorf("character limit = %d, want the 40 the call names", pageReader.call.CharacterLimit)
	}
	if pageReader.call.ToleratedAge != 30*time.Minute {
		t.Errorf(
			"tolerated age = %v, want the 30m the call names",
			pageReader.call.ToleratedAge,
		)
	}
}

func TestPageCallWithAnAddressThatCannotBeReadIsAnsweredAsAnError(t *testing.T) {
	session := openToolSession(t, &searchAnswering{}, &pageAnswering{page: pageAnswerUnderTest(t)})

	var answer any
	result := answerOf(t, session, "read_page", map[string]any{"url": "not a url"}, &answer)
	if !result.IsError {
		t.Error("the tool answered without an error, want the unreadable address to be an error")
	}
}

func TestPageCallWithAnAgeThatCannotBeReadIsAnsweredAsAnError(t *testing.T) {
	session := openToolSession(t, &searchAnswering{}, &pageAnswering{page: pageAnswerUnderTest(t)})

	var answer any
	result := answerOf(t, session, "read_page", map[string]any{
		"url":          pageAddress,
		"toleratedAge": "half an hour",
	}, &answer)
	if !result.IsError {
		t.Error("the tool answered without an error, want the unreadable age to be an error")
	}
}
