// Package http serves the markdown corpus contract.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

type PageMarkdownRecall interface {
	PageOf(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
	) (markdownrecall.PageUnderRequestedURL, bool, error)
}

type MarkdownRecallReceiver struct {
	recall PageMarkdownRecall
}

func NewMux(recall PageMarkdownRecall) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(http.MethodGet+" "+pagemarkdownstore.RecalledPagePath, New(recall))

	return mux
}

func New(recall PageMarkdownRecall) MarkdownRecallReceiver {
	return MarkdownRecallReceiver{recall: recall}
}

func (r MarkdownRecallReceiver) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestedURL, err := canonicalurl.CanonicalURLOf(
		request.URL.Query().Get(pagemarkdownstore.RequestedURLQueryParam),
	)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	page, held, err := r.recall.PageOf(request.Context(), requestedURL)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if !held {
		http.Error(
			writer,
			fmt.Sprintf("the corpus holds no markdown for %q", requestedURL),
			http.StatusNotFound,
		)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(recalledPageFrom(page))
}

func recalledPageFrom(page markdownrecall.PageUnderRequestedURL) pagemarkdownstore.RecalledPage {
	return pagemarkdownstore.RecalledPage{
		CanonicalURL: page.MarkdownURL.String(),
		Markdown:     string(page.Markdown),
		StoredAt:     page.StoredAt,
		Version:      page.Version,
	}
}
