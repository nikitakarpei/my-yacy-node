// Package leastreliable ranks the peer this deployment has had the least luck
// with as the stalest, so a full directory drops a peer that does not answer
// before a peer that has answered for weeks. Reliability belongs to a peer at
// one address while the directory keeps or drops the peer itself, so the
// address the peer has done best from speaks for it. A peer that was admitted too
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

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerReliability interface {
	ReliabilityOf(
		ctx context.Context,
		peerAtAddress peeranswerhistory.PeerAtAddress,
	) float64
}

type Source struct {
	reliability     PeerReliability
	refreshInterval time.Duration
	now             func() time.Time
}

func New(
	reliability PeerReliability,
	refreshInterval time.Duration,
	now func() time.Time,
) Source {
	return Source{
		reliability:     reliability,
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
			reliability:         s.reliabilityOf(ctx, peer),
			answeredAt:          peer.AnsweredAt,
			admittedAt:          peer.AdmittedAt,
		})
	}

	return ranked
}

func (s Source) reliabilityOf(ctx context.Context, peer peerdirectory.KnownPeer) float64 {
	bestReliability := 0.0
	for _, address := range peer.Addresses {
		bestReliability = max(bestReliability, s.reliability.ReliabilityOf(
			ctx, peeranswerhistory.PeerAtAddress{Hash: peer.Hash, Address: address},
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
