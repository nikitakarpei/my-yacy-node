package pagescrapecontract

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeRequest struct {
	PageURL           canonicalurl.CanonicalURL `json:"PageURL"`
	FetchURL          canonicalurl.CanonicalURL `json:"FetchURL,omitzero"`
	DeferredSince     time.Time                 `json:"DeferredSince,omitzero"`
	GivesUpOnDeferral bool                      `json:"GivesUpOnDeferral,omitzero"`
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
	if request.FetchURL == (canonicalurl.CanonicalURL{}) {
		request.FetchURL = request.PageURL
	}
	return request, nil
}
