package yacycrawlcontract

import (
	"encoding/json"
	"fmt"
	"time"
)

type PageContentRepresentation struct {
	CanonicalURL string
	Title        string
	CrawledAt    time.Time
	Language     string
	Format       PageFormat
	Body         []byte
}

func MarshalPageContentRepresentation(page PageContentRepresentation) ([]byte, error) {
	data, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("marshal page content representation: %w", err)
	}
	return data, nil
}

func UnmarshalPageContentRepresentation(data []byte) (PageContentRepresentation, error) {
	var page PageContentRepresentation
	if err := json.Unmarshal(data, &page); err != nil {
		return PageContentRepresentation{}, fmt.Errorf(
			"unmarshal page content representation: %w",
			err,
		)
	}
	return page, nil
}
