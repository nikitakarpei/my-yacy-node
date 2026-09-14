//go:build e2e

package yacypeer

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
)

// ResultLinksFor runs one global search on the real YaCy peer and returns the
// addresses it reports. verify=false parses to no cache strategy
// (CacheStrategy.java:76), so YaCy lists a result without fetching the document
// to build a snippet.
func ResultLinksFor(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL, query string,
) []string {
	t.Helper()

	searchURL := yacyURL + "/yacysearch.rss?" + url.Values{
		"query":          {query},
		"resource":       {"global"},
		"verify":         {"false"},
		"maximumRecords": {"10"},
	}.Encode()

	result := probe.Get(ctx, searchURL)
	if !result.OK {
		return nil
	}

	links, err := resultLinks([]byte(result.Body))
	if err != nil {
		t.Logf("yacysearch.rss body:\n%s", result.Body)
		t.Fatalf("read search results: %v", err)
	}

	return links
}

func resultLinks(body []byte) ([]string, error) {
	var feed struct {
		Links []string `xml:"channel>item>link"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse search feed: %w", err)
	}

	return feed.Links, nil
}
