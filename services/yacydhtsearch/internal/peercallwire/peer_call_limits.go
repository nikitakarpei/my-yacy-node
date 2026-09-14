package peercallwire

import "time"

type PeerCallLimits struct {
	MaxResponseBytes  int64
	PeerCallsInFlight int
	PeerCallBudget    time.Duration
}
