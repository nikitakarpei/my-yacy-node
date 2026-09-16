// Package peerreliability gives every peer one number that says how well this
// deployment's own probes have gone with it. A peer earns the number by staying
// reachable from one probe to the next, and loses it as its latest answer ages.
// The number belongs to the peer alone, so the same number orders the peers a
// query asks and the peers the directory keeps. Nothing here holds state or
// reads it.
package peerreliability

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
)

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

func (weights ReliabilityWeights) ReliabilityOf(
	observedPeer presenceaccrual.ObservedPeer,
	now time.Time,
) float64 {
	return weights.presenceBenefitOf(observedPeer) * weights.freshnessOf(observedPeer, now)
}

func (weights ReliabilityWeights) presenceBenefitOf(
	observedPeer presenceaccrual.ObservedPeer,
) float64 {
	if weights.MaturationDuration <= 0 {
		return 0
	}

	return min(
		float64(observedPeer.Presence)/float64(weights.MaturationDuration),
		1,
	)
}

func (weights ReliabilityWeights) freshnessOf(
	observedPeer presenceaccrual.ObservedPeer,
	now time.Time,
) float64 {
	if weights.StalenessHorizon <= 0 {
		return 0
	}
	sinceTheLatestAnswer := now.Sub(observedPeer.LatestAnsweredAt)
	if sinceTheLatestAnswer <= 0 {
		return 1
	}

	return max(1-float64(sinceTheLatestAnswer)/float64(weights.StalenessHorizon), 0)
}
