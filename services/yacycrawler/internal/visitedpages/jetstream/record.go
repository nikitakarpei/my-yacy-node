package jetstream

import (
	"encoding/json"
	"fmt"
	"time"
)

type pageVisit struct {
	VisitedAt  time.Time `json:"visitedAt"`
	EntityTag  string    `json:"entityTag,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

func marshalPageVisit(record pageVisit) ([]byte, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal page visit: %w", err)
	}
	return payload, nil
}

func unmarshalPageVisit(payload []byte) (pageVisit, error) {
	var record pageVisit
	if err := json.Unmarshal(payload, &record); err != nil {
		return pageVisit{}, fmt.Errorf("unmarshal page visit: %w", err)
	}
	return record, nil
}
