package firecrawlscrape_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/firecrawlscrape"
	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/markdownrecall"
)

type scrapeResponse struct {
	Success bool       `json:"success"`
	Data    scrapeData `json:"data"`
	Error   string     `json:"error"`
}

type scrapeData struct {
	Markdown string         `json:"markdown"`
	Metadata scrapeMetadata `json:"metadata"`
}

type scrapeMetadata struct {
	SourceURL string `json:"sourceURL"`
}

type fakeMarkdownRecaller struct {
	recalled     markdownrecall.RecalledPage
	failWith     error
	requestedURL string
}

func (f *fakeMarkdownRecaller) RecallPage(
	_ context.Context,
	requestedURL string,
) (markdownrecall.RecalledPage, error) {
	f.requestedURL = requestedURL
	return f.recalled, f.failWith
}

func serve(
	t *testing.T,
	recaller firecrawlscrape.MarkdownRecaller,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/v1/scrape", strings.NewReader(body),
	)
	recorder := httptest.NewRecorder()
	firecrawlscrape.NewScraper(recaller).ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) scrapeResponse {
	t.Helper()
	var response scrapeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func TestScrapeReturnsTheRecalledMarkdown(t *testing.T) {
	recaller := &fakeMarkdownRecaller{recalled: markdownrecall.RecalledPage{
		CanonicalURL: "https://example.com/",
		Markdown:     "# Hello",
	}}

	recorder := serve(t, recaller, `{"url":"https://example.com","formats":["markdown"]}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	response := decode(t, recorder)
	if !response.Success || response.Data.Markdown != "# Hello" {
		t.Errorf("response = %+v", response)
	}
	if response.Data.Metadata.SourceURL != "https://example.com/" {
		t.Errorf("sourceURL = %q", response.Data.Metadata.SourceURL)
	}
	if recaller.requestedURL != "https://example.com" {
		t.Errorf("recalled url = %q", recaller.requestedURL)
	}
}

func TestScrapeRejectsInvalidBody(t *testing.T) {
	recorder := serve(t, &fakeMarkdownRecaller{}, `not json`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if decode(t, recorder).Success {
		t.Errorf("success = true, want false")
	}
}

func TestScrapeRejectsMissingURL(t *testing.T) {
	recorder := serve(t, &fakeMarkdownRecaller{}, `{}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestScrapeReportsAnUnreachableCollaboratorAsAGatewayFailure(t *testing.T) {
	recaller := &fakeMarkdownRecaller{failWith: errors.New("boom")}

	recorder := serve(t, recaller, `{"url":"https://example.com"}`)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", recorder.Code)
	}
	response := decode(t, recorder)
	if response.Success || !strings.Contains(response.Error, "boom") {
		t.Errorf("response = %+v", response)
	}
}

func TestScrapeReportsMarkdownTheCorpusNeverHeldAsNotFound(t *testing.T) {
	recaller := &fakeMarkdownRecaller{
		failWith: fmt.Errorf("%w within the recall limit", markdownrecall.ErrMarkdownUnavailable),
	}

	recorder := serve(t, recaller, `{"url":"https://example.com"}`)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestScrapeReportsARecallBeyondTheInFlightLimitAsUnavailable(t *testing.T) {
	recaller := &fakeMarkdownRecaller{failWith: markdownrecall.ErrTooManyRecallsInFlight}

	recorder := serve(t, recaller, `{"url":"https://example.com"}`)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}
