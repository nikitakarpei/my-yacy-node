package jetstream

import (
	"encoding/json"
	"fmt"
	"time"
)

type lastPageVisit struct {
	VisitedAt  time.Time `json:"visitedAt"`
	EntityTag  string    `json:"entityTag,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

func marshalLastPageVisit(record lastPageVisit) ([]byte, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal last page visit: %w", err)
	}
	return payload, nil
}

func unmarshalLastPageVisit(payload []byte) (lastPageVisit, error) {
	var record lastPageVisit
	if err := json.Unmarshal(payload, &record); err != nil {
		return lastPageVisit{}, fmt.Errorf("unmarshal last page visit: %w", err)
	}
	return record, nil
}
