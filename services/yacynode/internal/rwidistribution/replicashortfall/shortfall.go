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
	Posting                      yacymodel.RWIPosting
	ReplicasNeeded               int
	HandoffReplicasNeeded        int
	Seeds                        []yacymodel.Seed
	RecipientsCloserThanThisNode []yacymodel.Hash
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

		offer, stale, err := r.replicasOf(ctx, posting, reachablePeers)
		if err != nil {
			return Due{}, err
		}
		result.Offers = append(result.Offers, offer)
		if len(stale.Peers) > 0 {
			result.Stale = append(result.Stale, stale)
		}
	}

	return result, nil
}

func (r *Shortfall) replicasOf(
	ctx context.Context,
	posting yacymodel.RWIPosting,
	reachablePeers []yacymodel.Seed,
) (ReplicaOffer, StaleReplicas, error) {
	holders, err := r.replicas.Holders(ctx, posting.WordHash, posting.URLHash)
	if err != nil {
		return ReplicaOffer{}, StaleReplicas{}, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.PostingPosition(posting.WordHash, posting.URLHash, r.partitions)
	acceptingPeers := peersAcceptingRemoteIndex(reachablePeers)
	replicasPeersOwe := r.replicasPeersOwe(acceptingPeers, position)
	responsibilityWindow := yacymodel.SeedsClosestToPosition(
		acceptingPeers,
		position,
		replicasPeersOwe,
	)
	held := r.holdersByResponsibility(
		ctx, holders, position, responsibilityWindow, replicasPeersOwe,
	)

	stale := StaleReplicas{
		Posting: postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash},
		Peers:   held.gone,
	}
	offer := ReplicaOffer{Posting: posting}
	if len(held.responsible) >= replicasPeersOwe {
		stale.Peers = append(stale.Peers, held.outsideWindow...)
		offer.Seeds = peersHoldingReplica(acceptingPeers, held.responsible)
		offer.HandoffReplicasNeeded = r.handoffReplicasNeeded(
			reachablePeers, held.responsible, position,
		)

		return offer, stale, nil
	}
	offer.ReplicasNeeded = replicasPeersOwe - len(held.responsible)
	offer.Seeds = yacymodel.SeedsClosestToPosition(
		peersWithoutReplica(
			peersEligibleForReplicas(acceptingPeers, r.eligibility),
			held.stillHolding(),
		),
		position,
		offer.ReplicasNeeded,
	)
	offer.HandoffReplicasNeeded = r.handoffReplicasNeeded(
		reachablePeers, held.stillHolding(), position,
	)
	offer.RecipientsCloserThanThisNode = peersCloserThanThisNode(offer.Seeds, position, r.self)

	return offer, stale, nil
}

func (r *Shortfall) handoffReplicasNeeded(
	reachablePeers []yacymodel.Seed,
	holders []yacymodel.Hash,
	position yacymodel.DHTPosition,
) int {
	closer := peersCloserThanThisNode(
		peersHoldingReplica(reachablePeers, holders), position, r.self,
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

// replicaHolders groups the peers holding a replica by the DHT responsibility
// that decides whether the ledger keeps their entry.
type replicaHolders struct {
	responsible   []yacymodel.Hash
	outsideWindow []yacymodel.Hash
	gone          []yacymodel.Hash
}

func (h replicaHolders) stillHolding() []yacymodel.Hash {
	return append(append([]yacymodel.Hash{}, h.responsible...), h.outsideWindow...)
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

func noFartherThanClosestPeers(
	peer yacymodel.Hash,
	position yacymodel.DHTPosition,
	closestPeers []yacymodel.Seed,
	replicasPeersOwe int,
) bool {
	if len(closestPeers) == 0 || len(closestPeers) < replicasPeersOwe {
		return true
	}
	farthest := closestPeers[len(closestPeers)-1].Hash

	return yacymodel.DistanceToPosition(peer, position) <=
		yacymodel.DistanceToPosition(farthest, position)
}
