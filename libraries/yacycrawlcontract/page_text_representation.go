package yacycrawlcontract

import (
	"encoding/json"
	"fmt"
)

type PageTextRepresentation struct {
	PageReference
	Text []byte `json:"Text"`
}

func MarshalPageTextRepresentation(page PageTextRepresentation) ([]byte, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("marshal page text representation: %w", err)
	}
	return data, nil
}

func UnmarshalPageTextRepresentation(data []byte) (PageTextRepresentation, error) {
	var page PageTextRepresentation
	if err := json.Unmarshal(data, &page); err != nil {
		return PageTextRepresentation{}, fmt.Errorf("unmarshal page text representation: %w", err)
	}
	return page, nil
}
