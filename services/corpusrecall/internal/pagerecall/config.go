package pagerecall

import "time"

type Config struct {
	Deadline     time.Duration
	PollInterval time.Duration
	MaxInFlight  int
}
