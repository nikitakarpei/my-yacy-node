// Package peersearchwire speaks one search to one peer address.
package peersearchwire

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const searchPath = "/yacy/search.html"

type PeerSearchObserver interface {
	PeerAnswered(
		ctx context.Context,
		address string,
		answeredItems []searchresult.Item,
		spent time.Duration,
	)
	PeerRefused(ctx context.Context, address string, status int, spent time.Duration)
	PeerUnreachable(ctx context.Context, address string, cause error, spent time.Duration)
	PeerAnswerUnreadable(ctx context.Context, address string, cause error, spent time.Duration)
}

type Wire struct {
	client           *http.Client
	maxResponseBytes int64
	observer         PeerSearchObserver
}

func New(client *http.Client, maxResponseBytes int64, observer PeerSearchObserver) Wire {
	return Wire{client: client, maxResponseBytes: maxResponseBytes, observer: observer}
}

func (w Wire) Search(
	ctx context.Context,
	address string,
	request yacyproto.SearchRequest,
) []searchresult.Item {
	startedAt := time.Now()
	body, ok := w.postSearch(ctx, address, request, startedAt)
	if !ok {
		return nil
	}

	response, err := yacyproto.ParseSearchResponse(ctx, yacyproto.ParseMessage(body))
	if err != nil {
		w.observer.PeerAnswerUnreadable(ctx, address, err, time.Since(startedAt))
		return nil
	}

	items := make([]searchresult.Item, 0, len(response.Resources))
	for _, resource := range response.Resources {
		item, ok := searchresult.ItemFrom(resource.Metadata)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	w.observer.PeerAnswered(ctx, address, items, time.Since(startedAt))

	return items
}

func (w Wire) postSearch(
	ctx context.Context,
	address string,
	request yacyproto.SearchRequest,
	startedAt time.Time,
) (string, bool) {
	form := request.Form().Encode()
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, address+searchPath, strings.NewReader(form),
	)
	if err != nil {
		w.observer.PeerUnreachable(ctx, address, err, time.Since(startedAt))
		return "", false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.client.Do(req)
	if err != nil {
		w.observer.PeerUnreachable(ctx, address, err, time.Since(startedAt))
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		w.observer.PeerRefused(ctx, address, resp.StatusCode, time.Since(startedAt))
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, w.maxResponseBytes))
	if err != nil {
		w.observer.PeerAnswerUnreadable(ctx, address, err, time.Since(startedAt))
		return "", false
	}

	return string(body), true
}

type PeerSearchObservers []PeerSearchObserver

func (observers PeerSearchObservers) PeerAnswered(
	ctx context.Context,
	address string,
	answeredItems []searchresult.Item,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnswered(ctx, address, answeredItems, spent)
	}
}

func (observers PeerSearchObservers) PeerRefused(
	ctx context.Context,
	address string,
	status int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerRefused(ctx, address, status, spent)
	}
}

func (observers PeerSearchObservers) PeerUnreachable(
	ctx context.Context,
	address string,
	cause error,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerUnreachable(ctx, address, cause, spent)
	}
}

func (observers PeerSearchObservers) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	cause error,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnswerUnreadable(ctx, address, cause, spent)
	}
}
