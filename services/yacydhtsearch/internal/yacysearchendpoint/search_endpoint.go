// Package yacysearchendpoint serves YaCy's public /yacysearch.json contract.
package yacysearchendpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

const (
	Path = "/yacysearch.json"

	fieldQuery          = "query"
	fieldStartRecord    = "startRecord"
	fieldMaximumRecords = "maximumRecords"
	fieldLanguage       = "lr"

	defaultMaximumRecords = 10
)

type Network interface {
	Search(
		ctx context.Context,
		query searchquery.Query,
	) (searchresult.Ranking, bool)
}

type SearchEndpoint struct {
	network Network
}

func NewMux(network Network) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(Path, New(network))

	return mux
}

func New(network Network) SearchEndpoint {
	return SearchEndpoint{network: network}
}

func (e SearchEndpoint) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	page := e.pageFor(request.Context(), request.URL.Query())

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(searchPageFrom(page))
}

func (e SearchEndpoint) pageFor(ctx context.Context, form url.Values) searchresult.Page {
	query := queryOf(form)
	if len(query.Words) == 0 {
		return searchresult.Page{}
	}
	ranking, _ := e.network.Search(ctx, query)

	return ranking.PageFrom(startRecordOf(form), maximumRecordsOf(form))
}

func queryOf(form url.Values) searchquery.Query {
	return queryreading.QueryFrom(form.Get(fieldQuery), form.Get(fieldLanguage))
}

func startRecordOf(form url.Values) int {
	return max(0, intOf(form.Get(fieldStartRecord), 0))
}

func maximumRecordsOf(form url.Values) int {
	return max(1, intOf(form.Get(fieldMaximumRecords), defaultMaximumRecords))
}

func intOf(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}
