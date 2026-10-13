package peereligibility

import "time"

type PauseCause string

const (
	PauseDeclined PauseCause = "peer_declined"
	PauseFailed   PauseCause = "peer_failed"
)

type Pause struct {
	EndsAt time.Time
	Cause  PauseCause
}
