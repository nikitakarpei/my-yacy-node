// Package peerwire posts a form-encoded request to a peer, trying the peer's base URLs
// in the given order until one gives a reply the caller accepts. Each attempt has a
// share of the time left, and a redirect ends it. It reports why each base URL failed.
package peerwire

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const responseMaxBodyBytes int64 = 1 << 20

const (
	failureReportMaxBytes int64 = 512
	failureReportAbsent         = "no reported body"
)

var (
	errReplyNotAccepted   = errors.New("reply not accepted")
	errAttemptTimeLimit   = errors.New("attempt time limit passed")
	errUnexpectedStatus   = errors.New("unexpected status")
	errRequestNotPrepared = errors.New("request not prepared")
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

type FormExchange struct {
	client *http.Client
	clock  Clock
}

func NewFormExchange(client *http.Client, clock Clock) FormExchange {
	peerClient := *client
	peerClient.CheckRedirect = refuseRedirect

	return FormExchange{client: &peerClient, clock: clock}
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (e FormExchange) Exchange(
	ctx context.Context,
	request Request,
	baseURLs []BaseURL,
	accepts func(yacyproto.ResponseFields) bool,
	budget time.Duration,
) (Answer, bool) {
	deadline := e.clock.Now().Add(budget)
	body := request.Form.Encode()
	var failures []attemptFailure
	for index, baseURL := range baseURLs {
		remaining := deadline.Sub(e.clock.Now())
		if remaining <= 0 {
			break
		}
		answer, err := e.try(
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
func (e FormExchange) try(
	ctx context.Context,
	path, body string,
	baseURL BaseURL,
	accepts func(yacyproto.ResponseFields) bool,
	limit time.Duration,
) (Answer, error) {
	attemptCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stop := e.clock.After(limit, func() { cancel(errAttemptTimeLimit) })
	defer stop()

	sentAt := e.clock.Now()
	fields, err := e.post(attemptCtx, baseURL, path, body)
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
		Time:    e.clock.Now().Sub(sentAt),
	}, nil
}

func (e FormExchange) post(
	ctx context.Context,
	baseURL BaseURL,
	path, body string,
) (yacyproto.ResponseFields, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL.urlOf(path),
		strings.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRequestNotPrepared, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post: %w", err)
	}
	defer closeResponseBody(ctx, resp.Body, path)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w %s: %s",
			errUnexpectedStatus,
			resp.Status,
			peerFailureReport(resp.Body),
		)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, responseMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read reply: %w", err)
	}

	return yacyproto.ParseResponseFields(string(raw)), nil
}

func closeResponseBody(ctx context.Context, body io.Closer, path string) {
	if err := body.Close(); err != nil {
		slog.WarnContext(
			ctx,
			"response body close failed",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
}

func peerFailureReport(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, failureReportMaxBytes))
	if err != nil && len(raw) == 0 {
		return failureReportAbsent
	}

	report := strings.Join(strings.Fields(string(raw)), " ")
	if report == "" {
		return failureReportAbsent
	}

	return report
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
