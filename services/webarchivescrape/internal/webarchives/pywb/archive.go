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

func (a *Archive) NewestCapturesFor(
	ctx context.Context,
	query Query,
	pageLimit int,
) (NewestCaptures, error) {
	if pageLimit < 0 {
		return NewestCaptures{}, fmt.Errorf("page limit must not be negative")
	}
	queryURL := a.queryURLOf(query)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return NewestCaptures{}, fmt.Errorf("build cdx query %s: %w", queryURL, err)
	}
	answer, err := a.client.Do(request)
	if err != nil {
		return NewestCaptures{}, fmt.Errorf("query cdx index %s: %w", queryURL, err)
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return NewestCaptures{}, fmt.Errorf("cdx index %s answered %s", queryURL, answer.Status)
	}
	return newestCapturesFrom(answer.Body, pageLimit)
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
	return parameters
}

func newestCapturesFrom(answerBody io.Reader, pageLimit int) (NewestCaptures, error) {
	rows := json.NewDecoder(answerBody)
	selection := newestCaptureSelection{}
	for {
		capture, exists, err := captureFrom(rows)
		if err != nil {
			return NewestCaptures{}, err
		}
		if !exists {
			return selection.complete(), nil
		}
		pageLimitReached, err := selection.add(capture, pageLimit)
		if err != nil {
			return NewestCaptures{}, err
		}
		if pageLimitReached {
			return selection.complete(), nil
		}
	}
}

func captureFrom(rows *json.Decoder) (Capture, bool, error) {
	var row captureRow
	if err := rows.Decode(&row); err != nil {
		if errors.Is(err, io.EOF) {
			return Capture{}, false, nil
		}
		return Capture{}, false, fmt.Errorf("read cdx row: %w", err)
	}
	return Capture(row), true, nil
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
