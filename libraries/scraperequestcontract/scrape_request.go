package scraperequestcontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeRequest struct {
	CanonicalURL canonicalurl.CanonicalURL `json:"CanonicalURL"`
}

func MarshalScrapeRequest(request ScrapeRequest) ([]byte, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal scrape request: %w", err)
	}
	return data, nil
}

func UnmarshalScrapeRequest(data []byte) (ScrapeRequest, error) {
	var request ScrapeRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return ScrapeRequest{}, fmt.Errorf("unmarshal scrape request: %w", err)
	}
	return request, nil
}
