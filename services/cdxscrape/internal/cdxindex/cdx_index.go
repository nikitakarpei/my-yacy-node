// Package cdxindex reads what a web archive holds. It is the only place that speaks the
// CDX server api, the index protocol that pywb, OpenWayback, and the Internet Archive
// answer alike. Capture is the vocabulary it yields.
package cdxindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

const (
	indexPathElement = "cdx"

	parameterURL       = "url"
	parameterOutput    = "output"
	parameterMatchType = "matchType"
	parameterFilter    = "filter"
	parameterFrom      = "from"
	parameterTo        = "to"
	parameterLimit     = "limit"

	outputJSON = "json"

	mediaTypeContainsFilter = "mime:"
	statusCodeEqualsFilter  = "=status:"
)

type Capture struct {
	URLKey      string
	Timestamp   string
	OriginalURL string
}

type Query struct {
	URL        string
	MatchType  string
	MediaType  string
	StatusCode int
	From       string
	To         string
	Limit      int
}

type CDXIndex struct {
	client     *http.Client
	cdxURL     *url.URL
	collection string
}

func New(client *http.Client, cdxURL *url.URL, collection string) *CDXIndex {
	return &CDXIndex{client: client, cdxURL: cdxURL, collection: collection}
}

func (i *CDXIndex) CapturesFor(ctx context.Context, query Query) ([]Capture, error) {
	queryURL := queryURLOf(i.cdxURL, i.collection, query)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build cdx query %s: %w", queryURL, err)
	}
	answer, err := i.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query cdx index %s: %w", queryURL, err)
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cdx index %s answered %s", queryURL, answer.Status)
	}
	return capturesFrom(answer.Body)
}

func queryURLOf(cdxURL *url.URL, collection string, query Query) *url.URL {
	queryURL := *cdxURL
	queryURL.Path = path.Join(cdxURL.Path, collection, indexPathElement)
	queryURL.RawQuery = parametersOf(query).Encode()
	return &queryURL
}

func parametersOf(query Query) url.Values {
	parameters := url.Values{}
	parameters.Set(parameterURL, query.URL)
	parameters.Set(parameterOutput, outputJSON)
	if query.MatchType != "" {
		parameters.Set(parameterMatchType, query.MatchType)
	}
	if query.MediaType != "" {
		parameters.Add(parameterFilter, mediaTypeContainsFilter+query.MediaType)
	}
	if query.StatusCode != 0 {
		parameters.Add(parameterFilter, statusCodeEqualsFilter+strconv.Itoa(query.StatusCode))
	}
	if query.From != "" {
		parameters.Set(parameterFrom, query.From)
	}
	if query.To != "" {
		parameters.Set(parameterTo, query.To)
	}
	if query.Limit != 0 {
		parameters.Set(parameterLimit, strconv.Itoa(query.Limit))
	}
	return parameters
}

func capturesFrom(answerBody io.Reader) ([]Capture, error) {
	captures := []Capture{}
	rows := json.NewDecoder(answerBody)
	for {
		var row captureRow
		if err := rows.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				return captures, nil
			}
			return nil, fmt.Errorf("read cdx row: %w", err)
		}
		captures = append(captures, Capture(row))
	}
}

type captureRow struct {
	URLKey      string `json:"urlkey"`
	Timestamp   string `json:"timestamp"`
	OriginalURL string `json:"url"`
}
