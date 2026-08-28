// Package pendingvisit names one URL a crawl order still owes a visit, and the
// frontier stream that carries it from the worker that found it to the worker
// that pays it.
package pendingvisit

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	StreamName = "YACY_CRAWL_FRONTIER"
	Subject    = "yacy.crawl.frontier"
)

type PendingVisit struct {
	OrderID string                    `json:"OrderID"`
	URL     canonicalurl.CanonicalURL `json:"URL"`
	Depth   int                       `json:"Depth"`
}

func MarshalPendingVisit(visit PendingVisit) ([]byte, error) {
	data, err := json.Marshal(visit)
	if err != nil {
		return nil, fmt.Errorf("marshal pending visit %s: %w", visit.URL, err)
	}
	return data, nil
}

func UnmarshalPendingVisit(data []byte) (PendingVisit, error) {
	var visit PendingVisit
	if err := json.Unmarshal(data, &visit); err != nil {
		return PendingVisit{}, fmt.Errorf("unmarshal pending visit: %w", err)
	}
	return visit, nil
}
