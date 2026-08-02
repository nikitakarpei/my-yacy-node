// Package replicashortfall works out, for each posting due an offer, which
// peers to offer it to under the DHT, how many of them must accept for the
// posting to be at redundancy, and which ledger entries the DHT no longer
// justifies.
package replicashortfall

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type ReplicaOffer struct {
	Posting        yacymodel.RWIPosting
	ReplicasNeeded int
	Seeds          []yacymodel.Seed
}

type StaleReplicas struct {
	Posting postingschedule.Identity
	Peers   []yacymodel.Hash
}

type Due struct {
	Offers []ReplicaOffer
	Stale  []StaleReplicas
	Gone   []postingschedule.Identity
}

type Reachability interface {
	Reachable(ctx context.Context, peer yacymodel.Hash) bool
	RecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool
}

// ReplicaEligibility reports whether a peer is able to receive a posting
// replica right now, independently of whether the peer is reachable at all.
type ReplicaEligibility interface {
	Eligible(peer yacymodel.Hash) bool
}

type Shortfall struct {
	schedule     *postingschedule.Schedule
	replicas     *postingreplicas.Replicas
	postings     rwipostings.PostingIndex
	reachability Reachability
	eligibility  ReplicaEligibility
	partitions   yacymodel.DHTRingPartitions
	self         yacymodel.Hash
	redundancy   int
}

//nolint:revive // argument-limit: eight explicit, independently-meaningful collaborators
func New(
	schedule *postingschedule.Schedule,
	replicas *postingreplicas.Replicas,
	postings rwipostings.PostingIndex,
	reachability Reachability,
	eligibility ReplicaEligibility,
	partitions yacymodel.DHTRingPartitions,
	self yacymodel.Hash,
	redundancy int,
) *Shortfall {
	return &Shortfall{
		schedule:     schedule,
		replicas:     replicas,
		postings:     postings,
		reachability: reachability,
		eligibility:  eligibility,
		partitions:   partitions,
		self:         self,
		redundancy:   redundancy,
	}
}

func (r *Shortfall) Due(
	ctx context.Context,
	limit int,
	reachablePeers []yacymodel.Seed,
) (Due, error) {
	due, err := r.schedule.DuePostings(ctx, limit)
	if err != nil {
		return Due{}, fmt.Errorf("read due postings: %w", err)
	}

	var result Due
	for _, identity := range due {
		posting, found, err := r.postings.Posting(ctx, identity.Word, identity.URL)
		if err != nil {
			return Due{}, fmt.Errorf("read posting: %w", err)
		}
		if !found {
			result.Gone = append(result.Gone, identity)

			continue
		}

		placement, err := r.placementOf(ctx, posting, reachablePeers)
		if err != nil {
			return Due{}, err
		}

		result.Offers = append(result.Offers, r.replicaOfferOf(placement))
		if stale := staleReplicasOf(placement); len(stale.Peers) > 0 {
			result.Stale = append(result.Stale, stale)
		}
	}

	return result, nil
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
		acceptingPeers: acceptingPeers,
		held: r.holdersByResponsibility(
			ctx, holders, position, responsibilityWindow, replicasPeersOwe,
		),
		replicasPeersOwe: replicasPeersOwe,
	}, nil
}

func (r *Shortfall) holdersByResponsibility(
	ctx context.Context,
	holders []yacymodel.Hash,
	position yacymodel.DHTPosition,
	responsibilityWindow []yacymodel.Seed,
	replicasPeersOwe int,
) replicaHolders {
	var held replicaHolders
	for _, peer := range holders {
		switch {
		case !r.reachability.Reachable(ctx, peer):
			if r.reachability.RecentlyReachable(ctx, peer) {
				held.responsible = append(held.responsible, peer)
			} else {
				held.gone = append(held.gone, peer)
			}
		case noFartherThanClosestPeers(peer, position, responsibilityWindow, replicasPeersOwe):
			held.responsible = append(held.responsible, peer)
		default:
			held.outsideWindow = append(held.outsideWindow, peer)
		}
	}

	return held
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

func (r *Shortfall) replicasPeersOwe(
	acceptingPeers []yacymodel.Seed,
	position yacymodel.DHTPosition,
) int {
	if len(peersCloserThanThisNode(acceptingPeers, position, r.self)) < r.redundancy {
		return r.redundancy - 1
	}

	return r.redundancy
}
