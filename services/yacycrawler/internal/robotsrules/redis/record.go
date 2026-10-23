package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
)

var errUnknownAvailability = errors.New("unknown robots.txt availability")

var availabilityNames = map[robotsexclusion.Availability]string{
	robotsexclusion.Stated:      "stated",
	robotsexclusion.Absent:      "absent",
	robotsexclusion.Unreachable: "unreachable",
}

type keptRules struct {
	Availability string        `json:"availability"`
	RobotsTxt    string        `json:"robotsTxt,omitempty"`
	Lifetime     time.Duration `json:"lifetime"`
}

func marshalRules(rules robotsexclusion.Rules) ([]byte, error) {
	payload, err := json.Marshal(keptRules{
		Availability: availabilityNames[rules.Availability],
		RobotsTxt:    rules.RobotsTxt,
		Lifetime:     rules.Lifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal robots rules: %w", err)
	}
	return payload, nil
}

func unmarshalRules(payload []byte) (robotsexclusion.Rules, error) {
	var record keptRules
	if err := json.Unmarshal(payload, &record); err != nil {
		return robotsexclusion.Rules{}, fmt.Errorf("unmarshal robots rules: %w", err)
	}
	availability, known := availabilityNamed(record.Availability)
	if !known {
		return robotsexclusion.Rules{}, fmt.Errorf(
			"unmarshal robots rules: %w %q", errUnknownAvailability, record.Availability,
		)
	}
	return robotsexclusion.Rules{
		Availability: availability,
		RobotsTxt:    record.RobotsTxt,
		Lifetime:     record.Lifetime,
	}, nil
}

func availabilityNamed(name string) (robotsexclusion.Availability, bool) {
	for availability, availabilityName := range availabilityNames {
		if availabilityName == name {
			return availability, true
		}
	}
	return robotsexclusion.Undetermined, false
}
