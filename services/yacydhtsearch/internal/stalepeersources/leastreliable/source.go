// Package leastreliable orders the peers of one admission, stalest first, so a
// full directory drops a peer that does not answer before a peer that has
// answered for weeks. It orders the peers the directory holds together with the
// peers a seedlist offers it, and reliability belongs to a peer at one address
// while the directory keeps or drops the peer itself, so the address the peer
// has done best from speaks for it. An offered peer stands before a held peer of
// the same reliability, so a directory with no room keeps what it has. A peer
// that was admitted too recently to have been probed is never the stalest, so
// every newcomer gets one refresh cycle to answer for itself before it can be
// dropped. Peers this deployment has never seen answer are ordered by the
// directory's own record of when they last answered.
package leastreliable

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerReliability interface {
	ReliabilityOf(
		ctx context.Context,
		peerAtAddress probeanswerhistory.PeerAtAddress,
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

func (s Source) StalestPeersFirst(
	ctx context.Context,
	members []peerdirectory.KnownPeer,
	candidates []peerdirectory.CandidatePeer,
) []yacymodel.Hash {
	ranked := slices.SortedFunc(
		slices.Values(s.rankedPeersFrom(ctx, members, candidates)),
		stalestFirst,
	)
	stalest := make([]yacymodel.Hash, 0, len(ranked))
	for _, peer := range ranked {
		stalest = append(stalest, peer.hash)
	}

	return stalest
}

func (s Source) rankedPeersFrom(
	ctx context.Context,
	members []peerdirectory.KnownPeer,
	candidates []peerdirectory.CandidatePeer,
) []rankedPeer {
	now := s.now()
	ranked := make([]rankedPeer, 0, len(members)+len(candidates))
	for _, member := range members {
		ranked = append(ranked, rankedPeer{
			hash:                member.Hash,
			heldByTheDirectory:  true,
			awaitsItsFirstProbe: now.Sub(member.AdmittedAt) < s.refreshInterval,
			reliability:         s.reliabilityOf(ctx, member.Hash, member.Addresses),
			answeredAt:          member.AnsweredAt,
			admittedAt:          member.AdmittedAt,
		})
	}
	for _, candidate := range candidates {
		ranked = append(ranked, rankedPeer{
			hash:        candidate.Hash,
			reliability: s.reliabilityOf(ctx, candidate.Hash, candidate.Addresses),
		})
	}

	return ranked
}

func (s Source) reliabilityOf(
	ctx context.Context,
	peer yacymodel.Hash,
	addresses []string,
) float64 {
	bestReliability := 0.0
	for _, address := range addresses {
		bestReliability = max(bestReliability, s.reliability.ReliabilityOf(
			ctx, probeanswerhistory.PeerAtAddress{Hash: peer, Address: address},
		))
	}

	return bestReliability
}

type rankedPeer struct {
	hash                yacymodel.Hash
	heldByTheDirectory  bool
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
		offeredPeersFirst(a, b),
		a.answeredAt.Compare(b.answeredAt),
		a.admittedAt.Compare(b.admittedAt),
	)
}

func offeredPeersFirst(a, b rankedPeer) int {
	if a.heldByTheDirectory == b.heldByTheDirectory {
		return 0
	}
	if a.heldByTheDirectory {
		return 1
	}

	return -1
}
