// Package firecrawlscrape serves the Firecrawl v1 scrape endpoint by recalling the
// markdown of a URL from the operator's own corpus.
package firecrawlscrape

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/internal/markdownrecall"
)

type MarkdownRecaller interface {
	RecallPage(ctx context.Context, requestedURL string) (markdownrecall.RecalledPage, error)
}

type Scraper struct {
	markdownRecaller MarkdownRecaller
}

func NewScraper(markdownRecaller MarkdownRecaller) *Scraper {
	return &Scraper{markdownRecaller: markdownRecaller}
}

func (s *Scraper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request scrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.fail(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if request.URL == "" {
		s.fail(w, r, http.StatusBadRequest, "url is required", nil)
		return
	}

	recalled, err := s.markdownRecaller.RecallPage(r.Context(), request.URL)
	if err != nil {
		s.failRecall(w, r, request.URL, err)
		return
	}

	slog.DebugContext(r.Context(), "scrape served", slog.String("url", request.URL))
	writeJSON(w, http.StatusOK, scrapeResponse{Success: true, Data: scrapeDataFrom(recalled)})
}

func (s *Scraper) failRecall(
	w http.ResponseWriter,
	r *http.Request,
	requestedURL string,
	err error,
) {
	slog.WarnContext(r.Context(), "recall failed",
		slog.String("url", requestedURL),
		slog.Any("error", err),
	)
	s.fail(w, r, recallFailureCodeOf(err), "recall failed", err)
}

func recallFailureCodeOf(err error) int {
	switch {
	case errors.Is(err, markdownrecall.ErrTooManyRecallsInFlight):
		return http.StatusServiceUnavailable
	case errors.Is(err, markdownrecall.ErrMarkdownUnavailable):
		return http.StatusNotFound
	default:
		return http.StatusBadGateway
	}
}

func (s *Scraper) fail(
	w http.ResponseWriter,
	r *http.Request,
	code int,
	message string,
	err error,
) {
	detail := message
	if err != nil {
		detail = message + ": " + err.Error()
	}
	slog.DebugContext(r.Context(), "scrape rejected", slog.String("error", detail))
	writeJSON(w, code, scrapeResponse{Success: false, Error: detail})
}

func scrapeDataFrom(recalled markdownrecall.RecalledPage) *scrapeData {
	data := &scrapeData{Markdown: recalled.Markdown}
	data.Metadata.SourceURL = recalled.CanonicalURL
	return data
}

func writeJSON(w http.ResponseWriter, code int, body scrapeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
