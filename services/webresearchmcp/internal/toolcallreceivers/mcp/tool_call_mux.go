// Package mcp serves the two research tools over the Model Context Protocol, on one HTTP
// endpoint that any assistant speaking that protocol can call. It turns a tool call into a
// call on the service, and the answer back into the shape the protocol carries.
package mcp

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const (
	ToolEndpointPath = "/mcp"

	serverName = "webresearchmcp"

	searchWebToolName        = "search_web"
	searchWebToolDescription = "Search the web and answer with the results of the operator's " +
		"own search engine, in the order the engine returns them."

	readPageToolName        = "read_page"
	readPageToolDescription = "Read one web page as markdown. The page is fetched first " +
		"unless the corpus already holds markdown the call accepts."
)

type WebSearch interface {
	SearchResultsFor(
		ctx context.Context,
		query string,
		searchResultLimit int,
	) ([]websearch.SearchResult, error)
}

type PageReader interface {
	PageAnswerFor(ctx context.Context, call pageread.PageCall) (pageread.PageAnswer, error)
}

type Config struct {
	Search              WebSearch
	PageReader          PageReader
	ServiceVersion      string
	ToolCallConcurrency int
}

func NewToolCallMux(cfg Config) *http.ServeMux {
	toolServer := toolServerFrom(cfg)
	mux := http.NewServeMux()
	mux.Handle(ToolEndpointPath, mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return toolServer },
		nil,
	))
	return mux
}

func toolServerFrom(cfg Config) *mcp.Server {
	admission := newToolCallAdmission(cfg.ToolCallConcurrency)
	toolServer := mcp.NewServer(
		&mcp.Implementation{Name: serverName, Version: cfg.ServiceVersion},
		nil,
	)
	webSearch := &webSearchTool{search: cfg.Search, admission: admission}
	mcp.AddTool(
		toolServer,
		&mcp.Tool{Name: searchWebToolName, Description: searchWebToolDescription},
		webSearch.searchWeb,
	)
	pageRead := &pageReadTool{pageReader: cfg.PageReader, admission: admission}
	mcp.AddTool(
		toolServer,
		&mcp.Tool{Name: readPageToolName, Description: readPageToolDescription},
		pageRead.readPage,
	)
	return toolServer
}
