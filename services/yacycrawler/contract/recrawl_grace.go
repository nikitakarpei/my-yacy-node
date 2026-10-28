package yacycrawlcontract

import (
	"encoding/json"
	"fmt"
	"time"
)

type RecrawlGrace struct {
	length time.Duration
}

var (
	NoRecrawlGrace  = RecrawlGrace{}
	MaxRecrawlGrace = RecrawlGrace{length: 30 * 24 * time.Hour}
)

func RecrawlGraceOf(length time.Duration) (RecrawlGrace, error) {
	if length < NoRecrawlGrace.length || length > MaxRecrawlGrace.length {
		return RecrawlGrace{}, fmt.Errorf(
			"recrawl grace %s is outside %s to %s",
			length, NoRecrawlGrace.length, MaxRecrawlGrace.length,
		)
	}
	return RecrawlGrace{length: length}, nil
}

func (grace RecrawlGrace) Length() time.Duration {
	return grace.length
}

func (grace RecrawlGrace) Covers(sinceLastVisit time.Duration) bool {
	return sinceLastVisit < grace.length
}

func (grace RecrawlGrace) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(grace.length)
	if err != nil {
		return nil, fmt.Errorf("marshal recrawl grace: %w", err)
	}
	return data, nil
}

func (grace *RecrawlGrace) UnmarshalJSON(data []byte) error {
	var length time.Duration
	if err := json.Unmarshal(data, &length); err != nil {
		return fmt.Errorf("unmarshal recrawl grace: %w", err)
	}
	parsed, err := RecrawlGraceOf(length)
	if err != nil {
		return err
	}
	*grace = parsed
	return nil
}
