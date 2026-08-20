package yacycrawlcontract

import (
	"encoding/json"
	"fmt"
)

type PageMarkdownRepresentation struct {
	PageReference
	Markdown []byte `json:"Markdown"`
}

func MarshalPageMarkdownRepresentation(page PageMarkdownRepresentation) ([]byte, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("marshal page markdown representation: %w", err)
	}
	return data, nil
}

func UnmarshalPageMarkdownRepresentation(data []byte) (PageMarkdownRepresentation, error) {
	var page PageMarkdownRepresentation
	if err := json.Unmarshal(data, &page); err != nil {
		return PageMarkdownRepresentation{}, fmt.Errorf(
			"unmarshal page markdown representation: %w",
			err,
		)
	}
	return page, nil
}
