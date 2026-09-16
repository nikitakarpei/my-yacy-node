package peerreliability

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
)

type PeerPresence interface {
	EarnedPresenceOf(
		ctx context.Context,
		peerAtAddress probeanswerhistory.PeerAtAddress,
	) time.Duration
	LatestAnswerOf(
		ctx context.Context,
		peerAtAddress probeanswerhistory.PeerAtAddress,
	) time.Time
}

type Reliability struct {
	presence PeerPresence
	weights  ReliabilityWeights
	now      func() time.Time
}

func New(
	presence PeerPresence,
	weights ReliabilityWeights,
	now func() time.Time,
) Reliability {
	return Reliability{presence: presence, weights: weights, now: now}
}

func (reliability Reliability) ReliabilityOf(
	ctx context.Context,
	peerAtAddress probeanswerhistory.PeerAtAddress,
) float64 {
	return reliability.presenceBenefitOf(
		reliability.presence.EarnedPresenceOf(ctx, peerAtAddress),
	) * reliability.freshnessOf(
		reliability.presence.LatestAnswerOf(ctx, peerAtAddress),
	)
}

func (reliability Reliability) presenceBenefitOf(earnedPresence time.Duration) float64 {
	if reliability.weights.MaturationDuration <= 0 {
		return 0
	}

	return min(float64(earnedPresence)/float64(reliability.weights.MaturationDuration), 1)
}

func (reliability Reliability) freshnessOf(latestAnswer time.Time) float64 {
	if reliability.weights.StalenessHorizon <= 0 {
		return 0
	}
	sinceTheLatestAnswer := reliability.now().Sub(latestAnswer)
	if sinceTheLatestAnswer <= 0 {
		return 1
	}

	return max(
		1-float64(sinceTheLatestAnswer)/float64(reliability.weights.StalenessHorizon),
		0,
	)
}
