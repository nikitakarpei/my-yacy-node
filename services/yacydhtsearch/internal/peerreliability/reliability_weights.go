// Package peerreliability gives every peer one number that says how well this
// deployment's own probes have gone with it. A peer earns the number by staying
// reachable from one probe to the next, and loses it as its latest answer ages.
// The number belongs to the peer at the address it answered on, so the same
// number orders the peers a query asks and the peers the directory keeps. The
// presence a peer has earned is read where it is kept; the durations that weigh
// it are held here.
package peerreliability

import "time"

type ReliabilityWeights struct {
	MaturationDuration time.Duration
	StalenessHorizon   time.Duration
}

func DefaultReliabilityWeights() ReliabilityWeights {
	return ReliabilityWeights{
		MaturationDuration: 7 * 24 * time.Hour,
		StalenessHorizon:   6 * time.Hour,
	}
}
