// Package pywb reads what a pywb instance holds and says where it replays a capture.
// Every rule here is pywb's own: the index path, the newline delimited answer it writes,
// the absence of a way to keep one capture per page, and the mp_ modifier.
//
// The mp_ modifier asks pywb for the page itself rather than for the frame it shows a
// reader, and for links and subresources that point back into the archive rather than at
// the origin. The url a capture was taken from ends the replay address, written as it was
// captured.
package pywb

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
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	indexPathElement     = "cdx"
	archivedPageModifier = "mp_"

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

type Archive struct {
	client     *http.Client
	pywbURL    *url.URL
	collection string
}

func New(client *http.Client, pywbURL *url.URL, collection string) *Archive {
	return &Archive{client: client, pywbURL: pywbURL, collection: collection}
}

func (a *Archive) CapturesFor(ctx context.Context, query Query) ([]Capture, error) {
	queryURL := a.queryURLOf(query)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build cdx query %s: %w", queryURL, err)
	}
	answer, err := a.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query cdx index %s: %w", queryURL, err)
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cdx index %s answered %s", queryURL, answer.Status)
	}
	return capturesFrom(answer.Body)
}

func (a *Archive) queryURLOf(query Query) *url.URL {
	queryURL := *a.pywbURL
	queryURL.Path = path.Join(a.pywbURL.Path, a.collection, indexPathElement)
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

func (a *Archive) ReplayURLOf(capture Capture) (canonicalurl.CanonicalURL, error) {
	replayURL := strings.Join(
		[]string{
			strings.TrimSuffix(a.pywbURL.String(), "/"),
			a.collection,
			capture.Timestamp + archivedPageModifier,
			capture.OriginalURL,
		},
		"/",
	)
	canonicalReplayURL, err := canonicalurl.CanonicalURLOf(replayURL)
	if err != nil {
		return canonicalurl.CanonicalURL{}, fmt.Errorf("read replay url %s: %w", replayURL, err)
	}
	return canonicalReplayURL, nil
}
