// Package pendingpagevisit names one URL a crawl still owes a page visit, as the
// frontier holds it from the worker that found it to the worker that pays it. A page
// links to visits one level deeper, and redirects to a visit at its own depth.
package pendingpagevisit

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type PendingPageVisit struct {
	CrawlID crawlid.CrawlID           `json:"CrawlID"`
	URL     canonicalurl.CanonicalURL `json:"URL"`
	Depth   int                       `json:"Depth"`
}

func (pageVisit PendingPageVisit) LinkedPageVisitTo(
	url canonicalurl.CanonicalURL,
) PendingPageVisit {
	return PendingPageVisit{CrawlID: pageVisit.CrawlID, URL: url, Depth: pageVisit.Depth + 1}
}

func (pageVisit PendingPageVisit) RedirectedPageVisitTo(
	redirectTarget canonicalurl.CanonicalURL,
) PendingPageVisit {
	return PendingPageVisit{CrawlID: pageVisit.CrawlID, URL: redirectTarget, Depth: pageVisit.Depth}
}

func MarshalPendingPageVisit(pageVisit PendingPageVisit) ([]byte, error) {
	data, err := json.Marshal(pageVisit)
	if err != nil {
		return nil, fmt.Errorf("marshal pending page visit %s: %w", pageVisit.URL, err)
	}
	return data, nil
}

func UnmarshalPendingPageVisit(data []byte) (PendingPageVisit, error) {
	var pageVisit PendingPageVisit
	if err := json.Unmarshal(data, &pageVisit); err != nil {
		return PendingPageVisit{}, fmt.Errorf("unmarshal pending page visit: %w", err)
	}
	return pageVisit, nil
}
