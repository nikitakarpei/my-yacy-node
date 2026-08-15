package firecrawlscrape_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/firecrawlscrape"
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
	Title     string `json:"title"`
	Language  string `json:"language"`
	SourceURL string `json:"sourceURL"`
}

type fakeRecaller struct {
	response *corpusrecallv1.RecallResponse
	err      error
	request  *corpusrecallv1.RecallRequest
}

func (f *fakeRecaller) Recall(
	_ context.Context,
	in *corpusrecallv1.RecallRequest,
	_ ...grpc.CallOption,
) (*corpusrecallv1.RecallResponse, error) {
	f.request = in
	return f.response, f.err
}

func serve(
	t *testing.T,
	recaller firecrawlscrape.Recaller,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	scraper := firecrawlscrape.NewScraper(recaller, time.Second)
	request := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/v1/scrape", strings.NewReader(body),
	)
	recorder := httptest.NewRecorder()
	scraper.ServeHTTP(recorder, request)
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

func TestScrapeReturnsMarkdown(t *testing.T) {
	recaller := &fakeRecaller{response: &corpusrecallv1.RecallResponse{
		Representations: []*corpusrecallv1.Representation{{
			Representation: &corpusrecallv1.Representation_Markdown{
				Markdown: &corpusrecallv1.MarkdownRepresentation{
					CanonicalUrl: "https://example.com/",
					Markdown:     "# Hello",
				},
			},
		}},
	}}

	recorder := serve(t, recaller, `{"url":"https://example.com"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	response := decode(t, recorder)
	if !response.Success {
		t.Errorf("success = false, want true")
	}
	if response.Data.Markdown != "# Hello" {
		t.Errorf("markdown = %q", response.Data.Markdown)
	}
	if response.Data.Metadata.SourceURL != "https://example.com/" {
		t.Errorf("sourceURL = %q", response.Data.Metadata.SourceURL)
	}
	kinds := recaller.request.GetKinds()
	if len(kinds) != 1 ||
		kinds[0] != corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN {
		t.Errorf("kinds = %v, want [MARKDOWN]", kinds)
	}
}

func TestScrapeMapsTextMetadata(t *testing.T) {
	recaller := &fakeRecaller{response: &corpusrecallv1.RecallResponse{
		Representations: []*corpusrecallv1.Representation{{
			Representation: &corpusrecallv1.Representation_Text{
				Text: &corpusrecallv1.TextRepresentation{
					CanonicalUrl: "https://example.com/",
					Title:        "Title",
					Language:     "en",
				},
			},
		}},
	}}

	recorder := serve(t, recaller, `{"url":"https://example.com","formats":["text"]}`)

	response := decode(t, recorder)
	if response.Data.Metadata.Title != "Title" || response.Data.Metadata.Language != "en" {
		t.Errorf("metadata = %+v", response.Data.Metadata)
	}
	if response.Data.Metadata.SourceURL != "https://example.com/" {
		t.Errorf("sourceURL = %q", response.Data.Metadata.SourceURL)
	}
	kinds := recaller.request.GetKinds()
	if len(kinds) != 1 ||
		kinds[0] != corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT {
		t.Errorf("kinds = %v, want [TEXT]", kinds)
	}
}

func TestScrapeUnknownFormatFallsBackToMarkdown(t *testing.T) {
	recaller := &fakeRecaller{response: &corpusrecallv1.RecallResponse{}}

	serve(t, recaller, `{"url":"https://example.com","formats":["screenshot"]}`)

	kinds := recaller.request.GetKinds()
	if len(kinds) != 1 ||
		kinds[0] != corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN {
		t.Errorf("kinds = %v, want [MARKDOWN]", kinds)
	}
}

func TestScrapeRejectsInvalidBody(t *testing.T) {
	recorder := serve(t, &fakeRecaller{}, `not json`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if decode(t, recorder).Success {
		t.Errorf("success = true, want false")
	}
}

func TestScrapeRejectsMissingURL(t *testing.T) {
	recorder := serve(t, &fakeRecaller{}, `{}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestScrapeReportsRecallFailure(t *testing.T) {
	recorder := serve(t, &fakeRecaller{err: errors.New("boom")}, `{"url":"https://example.com"}`)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", recorder.Code)
	}
	response := decode(t, recorder)
	if response.Success || !strings.Contains(response.Error, "boom") {
		t.Errorf("response = %+v", response)
	}
}
