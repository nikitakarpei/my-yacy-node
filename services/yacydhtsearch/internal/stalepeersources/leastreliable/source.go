// Package leastreliable ranks the peer this deployment has had the least luck
// with as the stalest, so a full directory drops a peer that does not answer
// before a peer that has answered for weeks. Presence belongs to a peer at one
// address while the directory keeps or drops the peer itself, so the address
// the peer has done best from speaks for it. A peer that was admitted too
// recently to have been probed is never the stalest, so every newcomer gets one
// refresh cycle to answer for itself before it can be dropped. Peers this
// deployment has never seen answer are ordered by the directory's own record of
// when they last answered.
package leastreliable

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerPresence interface {
	EarnedPresenceOf(
		ctx context.Context,
		peerAtAddress presenceaccrual.PeerAtAddress,
	) time.Duration
	LatestAnswerOf(
		ctx context.Context,
		peerAtAddress presenceaccrual.PeerAtAddress,
	) time.Time
}

type Source struct {
	presence        PeerPresence
	weights         peerreliability.ReliabilityWeights
	refreshInterval time.Duration
	now             func() time.Time
}

func New(
	presence PeerPresence,
	weights peerreliability.ReliabilityWeights,
	refreshInterval time.Duration,
	now func() time.Time,
) Source {
	return Source{
		presence:        presence,
		weights:         weights,
		refreshInterval: refreshInterval,
		now:             now,
	}
}

func (s Source) StalestPeers(
	ctx context.Context,
	known []peerdirectory.KnownPeer,
	limit int,
) []yacymodel.Hash {
	if limit <= 0 {
		return nil
	}
	ranked := slices.SortedFunc(
		slices.Values(s.rankedPeersFrom(ctx, known)),
		stalestFirst,
	)
	stalest := make([]yacymodel.Hash, 0, min(limit, len(ranked)))
	for _, peer := range ranked[:min(limit, len(ranked))] {
		stalest = append(stalest, peer.hash)
	}

	return stalest
}

func (s Source) rankedPeersFrom(
	ctx context.Context,
	known []peerdirectory.KnownPeer,
) []rankedPeer {
	now := s.now()
	ranked := make([]rankedPeer, 0, len(known))
	for _, peer := range known {
		ranked = append(ranked, rankedPeer{
			hash:                peer.Hash,
			awaitsItsFirstProbe: now.Sub(peer.AdmittedAt) < s.refreshInterval,
			reliability:         s.reliabilityOf(ctx, peer, now),
			answeredAt:          peer.AnsweredAt,
			admittedAt:          peer.AdmittedAt,
		})
	}

	return ranked
}

func (s Source) reliabilityOf(
	ctx context.Context,
	peer peerdirectory.KnownPeer,
	now time.Time,
) float64 {
	bestReliability := 0.0
	for _, address := range peer.Addresses {
		peerAtAddress := presenceaccrual.PeerAtAddress{Hash: peer.Hash, Address: address}
		bestReliability = max(bestReliability, s.weights.ReliabilityOf(
			s.presence.EarnedPresenceOf(ctx, peerAtAddress),
			s.presence.LatestAnswerOf(ctx, peerAtAddress),
			now,
		))
	}

	return bestReliability
}

type rankedPeer struct {
	hash                yacymodel.Hash
	awaitsItsFirstProbe bool
	reliability         float64
	answeredAt          time.Time
	admittedAt          time.Time
}

func stalestFirst(a, b rankedPeer) int {
	if a.awaitsItsFirstProbe != b.awaitsItsFirstProbe {
		if a.awaitsItsFirstProbe {
			return 1
		}

		return -1
	}

	return cmp.Or(
		cmp.Compare(a.reliability, b.reliability),
		a.answeredAt.Compare(b.answeredAt),
		a.admittedAt.Compare(b.admittedAt),
	)
}
