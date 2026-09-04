package yacycrawlcontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type CrawledPage struct {
	PageURL canonicalurl.CanonicalURL `json:"PageURL"`
}

func MarshalCrawledPage(page CrawledPage) ([]byte, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("marshal crawled page: %w", err)
	}
	return data, nil
}

func UnmarshalCrawledPage(data []byte) (CrawledPage, error) {
	var page CrawledPage
	if err := json.Unmarshal(data, &page); err != nil {
		return CrawledPage{}, fmt.Errorf("unmarshal crawled page: %w", err)
	}
	return page, nil
}
