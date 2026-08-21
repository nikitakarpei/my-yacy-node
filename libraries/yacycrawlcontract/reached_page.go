package yacycrawlcontract

import (
	"encoding/json"
	"fmt"
)

type ReachedPage struct {
	CanonicalURL string `json:"CanonicalURL"`
}

func MarshalReachedPage(page ReachedPage) ([]byte, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("marshal reached page: %w", err)
	}
	return data, nil
}

func UnmarshalReachedPage(data []byte) (ReachedPage, error) {
	var page ReachedPage
	if err := json.Unmarshal(data, &page); err != nil {
		return ReachedPage{}, fmt.Errorf("unmarshal reached page: %w", err)
	}
	return page, nil
}
