// Package firecrawlscrape serves the Firecrawl v1 scrape endpoint by recalling a
// URL's corpus representation from corpusrecall.
package firecrawlscrape

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

type Recaller interface {
	Recall(
		ctx context.Context,
		in *corpusrecallv1.RecallRequest,
		opts ...grpc.CallOption,
	) (*corpusrecallv1.RecallResponse, error)
}

type Scraper struct {
	recaller Recaller
	timeout  time.Duration
}

func NewScraper(recaller Recaller, timeout time.Duration) *Scraper {
	return &Scraper{recaller: recaller, timeout: timeout}
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

	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()

	recalled, err := s.recaller.Recall(ctx, &corpusrecallv1.RecallRequest{
		Url:   request.URL,
		Kinds: kindsFor(request.Formats),
	})
	if err != nil {
		slog.WarnContext(ctx, "recall failed",
			slog.String("url", request.URL),
			slog.Any("error", err),
		)
		s.fail(w, r, http.StatusBadGateway, "recall failed", err)
		return
	}

	slog.DebugContext(ctx, "scrape served", slog.String("url", request.URL))
	writeJSON(w, http.StatusOK, scrapeResponse{Success: true, Data: scrapeDataFrom(recalled)})
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

func kindsFor(formats []string) []corpusrecallv1.RepresentationKind {
	if len(formats) == 0 {
		return []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		}
	}
	kinds := make([]corpusrecallv1.RepresentationKind, 0, len(formats))
	for _, format := range formats {
		switch format {
		case "markdown":
			kinds = append(kinds, corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN)
		case "text":
			kinds = append(kinds, corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT)
		}
	}
	if len(kinds) == 0 {
		return []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		}
	}
	return kinds
}

func scrapeDataFrom(recalled *corpusrecallv1.RecallResponse) *scrapeData {
	data := &scrapeData{}
	for _, representation := range recalled.GetRepresentations() {
		if markdown := representation.GetMarkdown(); markdown != nil {
			data.Markdown = markdown.GetMarkdown()
			data.Metadata.SourceURL = markdown.GetCanonicalUrl()
		}
		if text := representation.GetText(); text != nil {
			data.Metadata.Title = text.GetTitle()
			data.Metadata.Language = text.GetLanguage()
			if data.Metadata.SourceURL == "" {
				data.Metadata.SourceURL = text.GetCanonicalUrl()
			}
		}
	}
	return data
}

func writeJSON(w http.ResponseWriter, code int, body scrapeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
