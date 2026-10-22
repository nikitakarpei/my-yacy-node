// Package http reaches yacycrawler through its operator API over HTTP. A refused order comes
// back as the crawler's reason, and a crawl the crawler does not know comes back as not found.
// A cancel or a pause that the crawl's state makes pointless changes nothing.
package http

import (
	"bytes"
	"cmp"
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

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	answerByteLimit  = 8 << 20
	refusalByteLimit = 4 << 10
)

var (
	errUnexpectedAnswer = errors.New("the crawler gave an unexpected answer")
	errNotFound         = errors.New("the crawler knows no such crawl")
)

type Crawler struct {
	apiURL string
	client *http.Client
}

func New(apiURL *url.URL, callDeadline time.Duration) *Crawler {
	return &Crawler{
		apiURL: strings.TrimSuffix(apiURL.String(), "/"),
		client: &http.Client{Timeout: callDeadline},
	}
}

func (crawler *Crawler) Place(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) (crawlID string, refusal string, err error) {
	answer, err := crawler.call(
		ctx,
		http.MethodPut,
		yacycrawlcontract.OrderPath.For(order.OrderID),
		order,
	)
	if err != nil {
		return "", "", err
	}
	defer closeBodyOf(answer)
	switch answer.StatusCode {
	case http.StatusOK, http.StatusCreated:
		var crawl yacycrawlcontract.Crawl
		if err := valueIn(answer, &crawl); err != nil {
			return "", "", err
		}
		return crawl.CrawlID, "", nil
	case http.StatusBadRequest:
		refusal, err := refusalIn(answer)
		return "", refusal, err
	default:
		return "", "", unexpectedAnswerError(answer)
	}
}

func (crawler *Crawler) CrawlsIn(
	ctx context.Context,
	states []yacycrawlcontract.CrawlState,
	cursor string,
	limit int,
) (yacycrawlcontract.CrawlPage, error) {
	query := url.Values{"limit": {strconv.Itoa(limit)}}
	for _, state := range states {
		query.Add("state", string(state))
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var page yacycrawlcontract.CrawlPage
	err := crawler.read(ctx, string(yacycrawlcontract.CrawlsPath)+"?"+query.Encode(), &page)
	return page, err
}

func (crawler *Crawler) CrawlOf(
	ctx context.Context,
	crawlID string,
) (yacycrawlcontract.Crawl, bool, error) {
	var crawl yacycrawlcontract.Crawl
	err := crawler.read(ctx, yacycrawlcontract.CrawlPath.For(crawlID), &crawl)
	if errors.Is(err, errNotFound) {
		return yacycrawlcontract.Crawl{}, false, nil
	}
	return crawl, err == nil, err
}

func (crawler *Crawler) HostsOf(
	ctx context.Context,
	crawlID string,
	cursor string,
	limit int,
) (yacycrawlcontract.CrawlHostPage, error) {
	query := url.Values{"limit": {strconv.Itoa(limit)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var page yacycrawlcontract.CrawlHostPage
	err := crawler.read(
		ctx,
		yacycrawlcontract.CrawlHostsPath.For(crawlID)+"?"+query.Encode(),
		&page,
	)
	return page, err
}

func (crawler *Crawler) CrawlerStatus(
	ctx context.Context,
) (yacycrawlcontract.CrawlerStatus, error) {
	var status yacycrawlcontract.CrawlerStatus
	err := crawler.read(ctx, string(yacycrawlcontract.CrawlerStatusPath), &status)
	return status, err
}

func (crawler *Crawler) Cancel(ctx context.Context, crawlID string) error {
	return crawler.change(ctx, http.MethodPost, yacycrawlcontract.CrawlCancelPath.For(crawlID), nil)
}

func (crawler *Crawler) Pause(ctx context.Context, crawlID string) error {
	return crawler.change(ctx, http.MethodPut, yacycrawlcontract.CrawlPausePath.For(crawlID),
		yacycrawlcontract.Pause{Paused: true})
}

func (crawler *Crawler) Resume(ctx context.Context, crawlID string) error {
	return crawler.change(ctx, http.MethodPut, yacycrawlcontract.CrawlPausePath.For(crawlID),
		yacycrawlcontract.Pause{Paused: false})
}

func (crawler *Crawler) PauseCrawler(ctx context.Context) error {
	return crawler.change(
		ctx,
		http.MethodPut,
		string(yacycrawlcontract.CrawlerPausePath),
		yacycrawlcontract.Pause{Paused: true},
	)
}

func (crawler *Crawler) ResumeCrawler(ctx context.Context) error {
	return crawler.change(
		ctx,
		http.MethodPut,
		string(yacycrawlcontract.CrawlerPausePath),
		yacycrawlcontract.Pause{Paused: false},
	)
}

func (crawler *Crawler) read(ctx context.Context, path string, value any) error {
	answer, err := crawler.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer closeBodyOf(answer)
	switch answer.StatusCode {
	case http.StatusOK:
		return valueIn(answer, value)
	case http.StatusNotFound:
		return errNotFound
	default:
		return unexpectedAnswerError(answer)
	}
}

func (crawler *Crawler) change(ctx context.Context, method string, path string, body any) error {
	answer, err := crawler.call(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer closeBodyOf(answer)
	switch answer.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound, http.StatusConflict:
		return nil
	default:
		return unexpectedAnswerError(answer)
	}
}

func (crawler *Crawler) call(
	ctx context.Context,
	method string,
	path string,
	body any,
) (*http.Response, error) {
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			return nil, fmt.Errorf("write the crawler request %s %s: %w", method, path, err)
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, crawler.apiURL+path, &payload)
	if err != nil {
		return nil, fmt.Errorf("build the crawler request %s %s: %w", method, path, err)
	}
	request.Header.Set("Content-Type", "application/json")
	answer, err := crawler.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call the crawler %s %s: %w", method, path, err)
	}
	return answer, nil
}

func closeBodyOf(answer *http.Response) {
	_ = answer.Body.Close()
}

func refusalIn(answer *http.Response) (string, error) {
	refusal, err := io.ReadAll(io.LimitReader(answer.Body, refusalByteLimit))
	if err != nil {
		return "", fmt.Errorf("read the crawler refusal: %w", err)
	}
	return cmp.Or(strings.TrimSpace(string(refusal)), answer.Status), nil
}

func unexpectedAnswerError(answer *http.Response) error {
	return fmt.Errorf("%w: %s %s answered %s", errUnexpectedAnswer,
		answer.Request.Method, answer.Request.URL.Path, answer.Status)
}

func valueIn(answer *http.Response, value any) error {
	if err := json.NewDecoder(io.LimitReader(answer.Body, answerByteLimit)).
		Decode(value); err != nil {
		return fmt.Errorf("read the crawler answer to %s: %w", answer.Request.URL.Path, err)
	}
	return nil
}
