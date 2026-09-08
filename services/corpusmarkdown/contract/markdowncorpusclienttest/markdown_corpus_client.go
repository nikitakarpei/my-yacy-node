// Package markdowncorpusclienttest calls the markdown corpus contract of a running service in a test.
package markdowncorpusclienttest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

type MarkdownCorpusClient struct {
	t             *testing.T
	recallAddress string
}

func New(t *testing.T, listenAddress string) MarkdownCorpusClient {
	t.Helper()
	return MarkdownCorpusClient{
		t:             t,
		recallAddress: "http://" + listenAddress + pagemarkdownstore.RecalledPagePath,
	}
}

func (c MarkdownCorpusClient) RecallPage(
	ctx context.Context,
	requestedURL string,
) (pagemarkdownstore.RecalledPage, int) {
	c.t.Helper()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.recallAddressOf(requestedURL),
		nil,
	)
	if err != nil {
		c.t.Fatalf("build the recall request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		c.t.Fatalf("recall page: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return pagemarkdownstore.RecalledPage{}, response.StatusCode
	}
	var recalled pagemarkdownstore.RecalledPage
	if err := json.NewDecoder(response.Body).Decode(&recalled); err != nil {
		c.t.Fatalf("decode the recalled page: %v", err)
	}
	return recalled, response.StatusCode
}

func (c MarkdownCorpusClient) recallAddressOf(requestedURL string) string {
	query := url.Values{pagemarkdownstore.RequestedURLQueryParam: {requestedURL}}
	return c.recallAddress + "?" + query.Encode()
}
