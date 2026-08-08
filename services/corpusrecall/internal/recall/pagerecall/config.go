package pagerecall

import "time"

type Config struct {
	RecallLimit         time.Duration
	PollInterval        time.Duration
	MaxRequestsInFlight int
}
