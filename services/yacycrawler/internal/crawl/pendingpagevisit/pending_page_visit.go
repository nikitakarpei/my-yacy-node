// Package pendingpagevisit names one URL a crawl order still owes a page visit, and the
// frontier stream that carries it from the worker that found it to the worker
// that pays it.
package pendingpagevisit

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	StreamName = "YACY_CRAWL_FRONTIER"
	Subject    = "yacy.crawl.frontier"
)

type PendingPageVisit struct {
	OrderID string                    `json:"OrderID"`
	URL     canonicalurl.CanonicalURL `json:"URL"`
	Depth   int                       `json:"Depth"`
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
