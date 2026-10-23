// Package http serves the search of the text corpus.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

type TextSearch interface {
	MatchesFor(
		ctx context.Context,
		search textsearch.Search,
	) ([]searchdocument.PageMatch, bool)
}

type TextSearchReceiver struct {
	textSearch TextSearch
}

func NewMux(textSearch TextSearch) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(
		http.MethodGet+" "+searchdocument.SearchPath,
		TextSearchReceiver{textSearch: textSearch},
	)

	return mux
}

func (r TextSearchReceiver) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	search, err := searchFrom(request.URL.Query())
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	matches, answered := r.textSearch.MatchesFor(request.Context(), search)
	if !answered {
		http.Error(writer, "the search index did not answer", http.StatusBadGateway)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(searchdocument.PageMatches{Matches: matches})
}

func searchFrom(params url.Values) (textsearch.Search, error) {
	offset, err := numberIn(params, searchdocument.OffsetParam, 0)
	if err != nil {
		return textsearch.Search{}, err
	}
	limit, err := numberIn(params, searchdocument.LimitParam, textsearch.DefaultMatchLimit)
	if err != nil {
		return textsearch.Search{}, err
	}
	return textsearch.SearchFrom(
		params.Get(searchdocument.SearchQueryParam),
		params.Get(searchdocument.LanguageParam),
		offset,
		limit,
	)
}

func numberIn(params url.Values, name string, defaultNumber int) (int, error) {
	if !params.Has(name) {
		return defaultNumber, nil
	}
	number, err := strconv.Atoi(params.Get(name))
	if err != nil {
		return 0, fmt.Errorf("%s %q: must be a whole number", name, params.Get(name))
	}
	return number, nil
}
