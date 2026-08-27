// Package pywb reads the newest replay addresses that a pywb instance holds.
// Every rule here is pywb's own: the index path, the ordered newline-delimited answer,
// the absence of a way to keep one capture per page, and the mp_ modifier.
//
// The mp_ modifier asks pywb for the page itself rather than for the frame it shows a
// reader, and for subresources that the archive serves rather than the origin. The url a
// capture was taken from ends the replay address, written as it was captured.
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

type CaptureQuery struct {
	URL        string
	MatchType  string
	MediaType  string
	StatusCode int
	From       string
	To         string
}

type NewestReplayURLs struct {
	ReplayURLs   []canonicalurl.CanonicalURL
	CapturesRead int
	HasMorePages bool
}

func New(client *http.Client, pywbURL *url.URL, collection string) *Archive {
	return &Archive{client: client, pywbURL: pywbURL, collection: collection}
}

func (a *Archive) NewestReplayURLsFor(
	ctx context.Context,
	queries []CaptureQuery,
	pageLimit int,
) (NewestReplayURLs, error) {
	captureSelection, err := a.captureSelectionFor(ctx, queries, pageLimit)
	if err != nil {
		return NewestReplayURLs{}, err
	}
	return a.newestReplayURLsFrom(captureSelection)
}

func (a *Archive) captureSelectionFor(
	ctx context.Context,
	queries []CaptureQuery,
	pageLimit int,
) (captureSelection, error) {
	if pageLimit < 0 {
		return captureSelection{}, fmt.Errorf("page limit must not be negative")
	}
	selection := joinedCaptureSelection{pageLimit: pageLimit}
	for _, query := range queries {
		if selection.pageLimitReached() {
			return selection.completeWithUnreadPages(), nil
		}
		querySelection, err := a.captureSelectionFromIndex(
			ctx,
			query,
			selection.remainingPageLimit(),
		)
		if err != nil {
			return captureSelection{}, err
		}
		selection.join(querySelection)
	}
	return selection.complete(), nil
}

func (a *Archive) captureSelectionFromIndex(
	ctx context.Context,
	query CaptureQuery,
	pageLimit int,
) (captureSelection, error) {
	queryURL := a.queryURLOf(query)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return captureSelection{}, fmt.Errorf("build cdx query %s: %w", queryURL, err)
	}
	answer, err := a.client.Do(request)
	if err != nil {
		return captureSelection{}, fmt.Errorf("query cdx index %s: %w", queryURL, err)
	}
	defer func() { _ = answer.Body.Close() }()
	if answer.StatusCode != http.StatusOK {
		return captureSelection{}, fmt.Errorf("cdx index %s answered %s", queryURL, answer.Status)
	}
	return captureSelectionFrom(answer.Body, pageLimit)
}

func (a *Archive) queryURLOf(query CaptureQuery) *url.URL {
	queryURL := *a.pywbURL
	queryURL.Path = path.Join(a.pywbURL.Path, a.collection, indexPathElement)
	queryURL.RawQuery = parametersOf(query).Encode()
	return &queryURL
}

func parametersOf(query CaptureQuery) url.Values {
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

func captureSelectionFrom(answerBody io.Reader, pageLimit int) (captureSelection, error) {
	rows := json.NewDecoder(answerBody)
	selection := newestCaptureSelection{}
	for {
		capture, exists, err := captureFrom(rows)
		if err != nil {
			return captureSelection{}, err
		}
		if !exists {
			return selection.complete(), nil
		}
		pageLimitReached, err := selection.add(capture, pageLimit)
		if err != nil {
			return captureSelection{}, err
		}
		if pageLimitReached {
			return selection.complete(), nil
		}
	}
}

func captureFrom(rows *json.Decoder) (capture, bool, error) {
	var row captureRow
	if err := rows.Decode(&row); err != nil {
		if errors.Is(err, io.EOF) {
			return capture{}, false, nil
		}
		return capture{}, false, fmt.Errorf("read cdx row: %w", err)
	}
	return capture(row), true, nil
}

type captureRow struct {
	URLKey      string `json:"urlkey"`
	Timestamp   string `json:"timestamp"`
	OriginalURL string `json:"url"`
}

func (a *Archive) newestReplayURLsFrom(
	captureSelection captureSelection,
) (NewestReplayURLs, error) {
	newestReplayURLs := NewestReplayURLs{
		ReplayURLs:   make([]canonicalurl.CanonicalURL, 0, len(captureSelection.captures)),
		CapturesRead: captureSelection.capturesRead,
		HasMorePages: captureSelection.hasMorePages,
	}
	for _, captured := range captureSelection.captures {
		replayURL, err := a.replayURLOf(captured)
		if err != nil {
			return NewestReplayURLs{}, err
		}
		newestReplayURLs.ReplayURLs = append(newestReplayURLs.ReplayURLs, replayURL)
	}
	return newestReplayURLs, nil
}

func (a *Archive) replayURLOf(captured capture) (canonicalurl.CanonicalURL, error) {
	replayURL := strings.Join(
		[]string{
			strings.TrimSuffix(a.pywbURL.String(), "/"),
			a.collection,
			captured.Timestamp + archivedPageModifier,
			captured.OriginalURL,
		},
		"/",
	)
	canonicalReplayURL, err := canonicalurl.CanonicalURLOf(replayURL)
	if err != nil {
		return canonicalurl.CanonicalURL{}, fmt.Errorf("read replay url %s: %w", replayURL, err)
	}
	return canonicalReplayURL, nil
}
