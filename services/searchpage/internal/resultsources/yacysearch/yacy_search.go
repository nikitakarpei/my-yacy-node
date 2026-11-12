// Package yacysearch asks a YaCy search server for the ranking of a search down to the depth of its page, over YaCy's
// public yacysearch.json contract. A search that fails or gets an answer it cannot read is
// reported to the observer and gives no ranking.
package yacysearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

const (
	searchPath          = "/yacysearch.json"
	fieldQuery          = "query"
	fieldMaximumRecords = "maximumRecords"
	fieldLanguage       = "lr"
	languagePrefix      = "lang_"
	answerByteLimit     = 1 << 20
)

var errUnexpectedAnswer = errors.New("the YaCy search server gave an unexpected answer")

type Observer interface {
	SearchFailed(ctx context.Context, err error)
}

type YaCySearch struct {
	observer  Observer
	client    *http.Client
	searchURL string
}

type searchPageWireForm struct {
	Channels []struct {
		Items []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Description string `json:"description"`
		} `json:"items"`
	} `json:"channels"`
}

func New(
	serverURL *url.URL,
	dial func(ctx context.Context, network, address string) (net.Conn, error),
	callDeadline time.Duration,
	observer Observer,
) *YaCySearch {
	return &YaCySearch{
		observer: observer,
		client: &http.Client{
			Timeout:   callDeadline,
			Transport: &http.Transport{DialContext: dial},
		},
		searchURL: strings.TrimSuffix(serverURL.String(), "/") + searchPath,
	}
}

func (y *YaCySearch) RankingFor(
	ctx context.Context,
	search searchqueries.Search,
) (searchresults.SourceRanking, bool) {
	wireForm, err := y.searchPageFor(ctx, search)
	if err != nil {
		y.observer.SearchFailed(ctx, err)
		return nil, false
	}
	return sourceRankingOf(wireForm), true
}

func (y *YaCySearch) searchPageFor(
	ctx context.Context,
	search searchqueries.Search,
) (searchPageWireForm, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, y.searchURL+"?"+searchFormOf(search).Encode(), nil,
	)
	if err != nil {
		return searchPageWireForm{}, fmt.Errorf("build the search: %w", err)
	}
	answer, err := y.client.Do(request)
	if err != nil {
		return searchPageWireForm{}, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return searchPageWireForm{}, fmt.Errorf(
			"%w: status %d", errUnexpectedAnswer, answer.StatusCode,
		)
	}
	var wireForm searchPageWireForm
	if err := json.NewDecoder(io.LimitReader(answer.Body, answerByteLimit)).
		Decode(&wireForm); err != nil {
		return searchPageWireForm{}, fmt.Errorf("%w: %w", errUnexpectedAnswer, err)
	}
	return wireForm, nil
}

func searchFormOf(search searchqueries.Search) url.Values {
	form := url.Values{
		fieldQuery:          {search.Text()},
		fieldMaximumRecords: {strconv.Itoa(searchresults.DepthOf(search.PageNumber))},
	}
	if search.Language != "" {
		form.Set(fieldLanguage, languagePrefix+search.Language)
	}
	return form
}

func sourceRankingOf(wireForm searchPageWireForm) searchresults.SourceRanking {
	var ranking searchresults.SourceRanking
	for _, channel := range wireForm.Channels {
		for _, item := range channel.Items {
			ranking = append(ranking, searchresults.Result{
				Title:   item.Title,
				Address: item.Link,
				Snippet: item.Description,
			})
		}
	}
	return ranking
}
