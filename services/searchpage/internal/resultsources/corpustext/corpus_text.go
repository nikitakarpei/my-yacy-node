// Package corpustext asks corpustext for one result page of a search, over the HTTP search of
// its contract. A search that fails or gets an answer it cannot read is reported to the
// observer and gives no page.
package corpustext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

const answerByteLimit = 1 << 20

var errUnexpectedAnswer = errors.New("corpustext gave an unexpected answer")

type Observer interface {
	SearchFailed(ctx context.Context, err error)
}

type CorpusText struct {
	observer  Observer
	client    *http.Client
	searchURL string
}

func New(serviceURL *url.URL, callDeadline time.Duration, observer Observer) *CorpusText {
	return &CorpusText{
		observer:  observer,
		client:    &http.Client{Timeout: callDeadline},
		searchURL: strings.TrimSuffix(serviceURL.String(), "/") + searchdocument.SearchPath,
	}
}

func (c *CorpusText) PageFor(
	ctx context.Context,
	search searchqueries.Search,
) (searchresults.SourcePage, bool) {
	matches, err := c.matchesFor(ctx, search)
	if err != nil {
		c.observer.SearchFailed(ctx, err)
		return nil, false
	}
	return sourcePageOf(matches), true
}

func (c *CorpusText) matchesFor(
	ctx context.Context,
	search searchqueries.Search,
) (searchdocument.PageMatches, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.searchURL+"?"+searchFormOf(search).Encode(), nil,
	)
	if err != nil {
		return searchdocument.PageMatches{}, fmt.Errorf("build the search: %w", err)
	}
	answer, err := c.client.Do(request)
	if err != nil {
		return searchdocument.PageMatches{}, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return searchdocument.PageMatches{}, fmt.Errorf(
			"%w: status %d", errUnexpectedAnswer, answer.StatusCode,
		)
	}
	var matches searchdocument.PageMatches
	if err := json.NewDecoder(io.LimitReader(answer.Body, answerByteLimit)).
		Decode(&matches); err != nil {
		return searchdocument.PageMatches{}, fmt.Errorf("%w: %w", errUnexpectedAnswer, err)
	}
	return matches, nil
}

func searchFormOf(search searchqueries.Search) url.Values {
	return url.Values{
		searchdocument.SearchQueryParam: {search.Text()},
		searchdocument.LanguageParam:    {search.Language},
		searchdocument.OffsetParam:      {strconv.Itoa(searchresults.OffsetOf(search.PageNumber))},
		searchdocument.LimitParam:       {strconv.Itoa(searchresults.ResultsPerSourcePage)},
	}
}

func sourcePageOf(matches searchdocument.PageMatches) searchresults.SourcePage {
	page := make(searchresults.SourcePage, 0, len(matches.Matches))
	for _, match := range matches.Matches {
		page = append(page, searchresults.Result{
			Title:   match.Title,
			Address: match.URL,
			Snippet: match.Snippet,
		})
	}
	return page
}
