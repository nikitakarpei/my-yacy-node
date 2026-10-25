// Package peerwire posts a form-encoded request to a peer, trying the peer's base URLs
// in the given order until one gives a reply the caller accepts. Each attempt has a
// share of the time left, and a redirect ends it. It reports why each base URL failed.
package peerwire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

var (
	errReplyNotAccepted = errors.New("reply not accepted")
	errAttemptTimeLimit = errors.New("attempt time limit passed")
)

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

type Request struct {
	Peer yacymodel.Hash
	Path string
	Form url.Values
}

type Answer struct {
	Fields  yacyproto.ResponseFields
	BaseURL BaseURL
	Time    time.Duration
}

type attemptFailure struct {
	baseURL BaseURL
	cause   error
}

type Walker struct {
	poster formPoster
	clock  Clock
}

func NewWalker(client *http.Client, clock Clock) Walker {
	return Walker{poster: newFormPoster(client), clock: clock}
}

func (w Walker) TryInTurn(
	ctx context.Context,
	request Request,
	baseURLs []BaseURL,
	accepts func(yacyproto.ResponseFields) bool,
	budget time.Duration,
) (Answer, bool) {
	deadline := w.clock.Now().Add(budget)
	body := request.Form.Encode()
	var failures []attemptFailure
	for index, baseURL := range baseURLs {
		remaining := deadline.Sub(w.clock.Now())
		if remaining <= 0 {
			break
		}
		answer, err := w.try(
			ctx,
			request.Path,
			body,
			baseURL,
			accepts,
			remaining/time.Duration(len(baseURLs)-index),
		)
		if err == nil {
			reportPassedOver(ctx, request, failures)

			return answer, true
		}
		failures = append(failures, attemptFailure{baseURL: baseURL, cause: err})
	}
	reportUnanswered(ctx, request, failures)

	return Answer{}, false
}

//nolint:revive // argument-limit: one attempt of the walk, each argument a distinct part of it
func (w Walker) try(
	ctx context.Context,
	path, body string,
	baseURL BaseURL,
	accepts func(yacyproto.ResponseFields) bool,
	limit time.Duration,
) (Answer, error) {
	attemptCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stop := w.clock.After(limit, func() { cancel(errAttemptTimeLimit) })
	defer stop()

	sentAt := w.clock.Now()
	fields, err := w.poster.post(attemptCtx, baseURL, path, body)
	if err != nil {
		if cause := context.Cause(attemptCtx); cause != nil {
			return Answer{}, fmt.Errorf("attempt ended: %w", cause)
		}

		return Answer{}, err
	}
	if !accepts(fields) {
		return Answer{}, errReplyNotAccepted
	}

	return Answer{
		Fields:  fields,
		BaseURL: baseURL,
		Time:    w.clock.Now().Sub(sentAt),
	}, nil
}

func reportPassedOver(ctx context.Context, request Request, failures []attemptFailure) {
	for _, failure := range failures {
		slog.DebugContext(ctx, "peer base URL passed over",
			slog.String("peer", request.Peer.String()),
			slog.String("baseUrl", failure.baseURL.String()),
			slog.String("path", request.Path),
			slog.Any("cause", failure.cause),
		)
	}
}

func reportUnanswered(ctx context.Context, request Request, failures []attemptFailure) {
	for _, failure := range failures {
		slog.WarnContext(ctx, "peer base URL did not answer",
			slog.String("peer", request.Peer.String()),
			slog.String("baseUrl", failure.baseURL.String()),
			slog.String("path", request.Path),
			slog.Any("cause", failure.cause),
		)
	}
}
