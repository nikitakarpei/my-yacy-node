package peercallwire

import "time"

type PeerCallLimits struct {
	MaxResponseBytes         int64
	PeerCallsInFlight        int
	URLMetadataCallBudget    time.Duration
	SearchCallBudget         time.Duration
	SearchCallHeadersTimeout time.Duration
}
