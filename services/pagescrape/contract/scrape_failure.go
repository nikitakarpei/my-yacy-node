package pagescrapecontract

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeFailureReason string

const (
	NotModified           ScrapeFailureReason = "not-modified"
	AccessRefused         ScrapeFailureReason = "access-refused"
	RedirectsExhausted    ScrapeFailureReason = "redirects-exhausted"
	RedirectTargetInvalid ScrapeFailureReason = "redirect-target-invalid"
	Oversized             ScrapeFailureReason = "oversized"
	NoReasonGiven         ScrapeFailureReason = "no-reason-given"
	Deferred              ScrapeFailureReason = "deferred"
	DeferredTooLong       ScrapeFailureReason = "deferred-too-long"
)

type ScrapeFailure struct {
	PageURL  canonicalurl.CanonicalURL `json:"PageURL"`
	FetchURL canonicalurl.CanonicalURL `json:"FetchURL"`
	Reason   ScrapeFailureReason       `json:"Reason"`
}

func MarshalScrapeFailure(failure ScrapeFailure) ([]byte, error) {
	data, err := json.Marshal(failure)
	if err != nil {
		return nil, fmt.Errorf("marshal scrape failure: %w", err)
	}
	return data, nil
}

func UnmarshalScrapeFailure(data []byte) (ScrapeFailure, error) {
	var failure ScrapeFailure
	if err := json.Unmarshal(data, &failure); err != nil {
		return ScrapeFailure{}, fmt.Errorf("unmarshal scrape failure: %w", err)
	}
	if failure.Reason == "" {
		return ScrapeFailure{}, fmt.Errorf("unmarshal scrape failure: no reason")
	}
	return failure, nil
}
