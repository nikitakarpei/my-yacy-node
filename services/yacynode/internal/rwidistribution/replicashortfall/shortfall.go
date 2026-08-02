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
	redundancy   int
}

//nolint:revive // argument-limit: seven explicit, independently-meaningful collaborators
func New(
	schedule *postingschedule.Schedule,
	replicas *postingreplicas.Replicas,
	postings rwipostings.PostingIndex,
	reachability Reachability,
	eligibility ReplicaEligibility,
	partitions yacymodel.DHTRingPartitions,
	redundancy int,
) *Shortfall {
	return &Shortfall{
		schedule:     schedule,
		replicas:     replicas,
		postings:     postings,
		reachability: reachability,
		eligibility:  eligibility,
		partitions:   partitions,
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
	responsibilityWindow := yacymodel.SeedsClosestToPosition(
		acceptingPeers,
		position,
		r.redundancy,
	)
	held := r.holdersByResponsibility(ctx, holders, position, responsibilityWindow)

	stale := StaleReplicas{
		Posting: postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash},
		Peers:   held.gone,
	}
	offer := ReplicaOffer{Posting: posting}
	if len(held.responsible) >= r.redundancy {
		stale.Peers = append(stale.Peers, held.outsideWindow...)
		offer.Seeds = peersHoldingReplica(acceptingPeers, held.responsible)

		return offer, stale, nil
	}
	offer.ReplicasNeeded = r.redundancy - len(held.responsible)
	offer.Seeds = yacymodel.SeedsClosestToPosition(
		peersWithoutReplica(
			peersEligibleForReplicas(acceptingPeers, r.eligibility),
			held.stillHolding(),
		),
		position,
		offer.ReplicasNeeded,
	)

	return offer, stale, nil
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
		case r.noFartherThanClosestPeers(peer, position, responsibilityWindow):
			held.responsible = append(held.responsible, peer)
		default:
			held.outsideWindow = append(held.outsideWindow, peer)
		}
	}

	return held
}

func (r *Shortfall) noFartherThanClosestPeers(
	peer yacymodel.Hash,
	position yacymodel.DHTPosition,
	closestPeers []yacymodel.Seed,
) bool {
	if len(closestPeers) < r.redundancy {
		return true
	}
	farthest := closestPeers[len(closestPeers)-1].Hash

	return yacymodel.DistanceToPosition(peer, position) <=
		yacymodel.DistanceToPosition(farthest, position)
}
