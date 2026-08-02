package replicashortfall

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
)

// replicaPlacement is where one posting's replicas stand against the DHT at the
// moment the cycle reads it.
type replicaPlacement struct {
	posting          yacymodel.RWIPosting
	identity         postingschedule.Identity
	position         yacymodel.DHTPosition
	reachablePeers   []yacymodel.Seed
	acceptingPeers   []yacymodel.Seed
	held             replicaHolders
	replicasPeersOwe int
}

func (p replicaPlacement) atRedundancy() bool {
	return len(p.held.responsible) >= p.replicasPeersOwe
}

func (r *Shortfall) placementOf(
	ctx context.Context,
	posting yacymodel.RWIPosting,
	reachablePeers []yacymodel.Seed,
) (replicaPlacement, error) {
	holders, err := r.replicas.Holders(ctx, posting.WordHash, posting.URLHash)
	if err != nil {
		return replicaPlacement{}, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.PostingPosition(posting.WordHash, posting.URLHash, r.partitions)
	acceptingPeers := peersAcceptingRemoteIndex(reachablePeers)
	replicasPeersOwe := r.replicasPeersOwe(acceptingPeers, position)
	responsibilityWindow := yacymodel.SeedsClosestToPosition(
		acceptingPeers,
		position,
		replicasPeersOwe,
	)

	return replicaPlacement{
		posting:        posting,
		identity:       postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash},
		position:       position,
		reachablePeers: reachablePeers,
		acceptingPeers: acceptingPeers,
		held: r.holdersByResponsibility(
			ctx, holders, position, responsibilityWindow, replicasPeersOwe,
		),
		replicasPeersOwe: replicasPeersOwe,
	}, nil
}

func (r *Shortfall) replicaOfferOf(placement replicaPlacement) ReplicaOffer {
	offer := ReplicaOffer{Posting: placement.posting}
	if placement.atRedundancy() {
		offer.Seeds = peersHoldingReplica(placement.acceptingPeers, placement.held.responsible)

		return offer
	}

	offer.ReplicasNeeded = placement.replicasPeersOwe - len(placement.held.responsible)
	offer.Seeds = yacymodel.SeedsClosestToPosition(
		peersWithoutReplica(
			peersEligibleForReplicas(placement.acceptingPeers, r.eligibility),
			placement.held.stillHolding(),
		),
		placement.position,
		offer.ReplicasNeeded,
	)

	return offer
}

func (r *Shortfall) replicaHandoffOf(
	placement replicaPlacement,
	offer ReplicaOffer,
) ReplicaHandoff {
	if placement.atRedundancy() {
		return ReplicaHandoff{
			Posting:        placement.identity,
			ReplicasNeeded: r.handoffReplicasNeeded(placement, placement.held.responsible),
		}
	}

	return ReplicaHandoff{
		Posting:        placement.identity,
		ReplicasNeeded: r.handoffReplicasNeeded(placement, placement.held.stillHolding()),
		Peers:          peersCloserThanThisNode(offer.Seeds, placement.position, r.self),
	}
}

func staleReplicasOf(placement replicaPlacement) StaleReplicas {
	stale := StaleReplicas{Posting: placement.identity, Peers: placement.held.gone}
	if placement.atRedundancy() {
		stale.Peers = append(stale.Peers, placement.held.outsideWindow...)
	}

	return stale
}

func (r *Shortfall) handoffReplicasNeeded(
	placement replicaPlacement,
	holders []yacymodel.Hash,
) int {
	closer := peersCloserThanThisNode(
		peersHoldingReplica(placement.reachablePeers, holders),
		placement.position,
		r.self,
	)

	return r.redundancy - len(closer)
}

func (r *Shortfall) replicasPeersOwe(
	acceptingPeers []yacymodel.Seed,
	position yacymodel.DHTPosition,
) int {
	if len(peersCloserThanThisNode(acceptingPeers, position, r.self)) < r.redundancy {
		return r.redundancy - 1
	}

	return r.redundancy
}
