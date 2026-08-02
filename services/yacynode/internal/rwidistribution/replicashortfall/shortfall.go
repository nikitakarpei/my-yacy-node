// Package replicashortfall works out, for each posting due an offer, which
// peers should hold a replica under the DHT, how many replicas are still missing,
// and which ledger entries the DHT no longer justifies.
package replicashortfall

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type MissingReplicas struct {
	Posting        yacymodel.RWIPosting
	ReplicasNeeded int
	Seeds          []yacymodel.Seed
}

type StaleReplicas struct {
	Posting postingschedule.Identity
	Peers   []yacymodel.Hash
}

type Due struct {
	Missing []MissingReplicas
	Stale   []StaleReplicas
	Gone    []postingschedule.Identity
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

		missing, stale, err := r.replicasOf(ctx, posting, reachablePeers)
		if err != nil {
			return Due{}, err
		}
		result.Missing = append(result.Missing, missing)
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
) (MissingReplicas, StaleReplicas, error) {
	holders, err := r.replicas.Holders(ctx, posting.WordHash, posting.URLHash)
	if err != nil {
		return MissingReplicas{}, StaleReplicas{}, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.PostingPosition(posting.WordHash, posting.URLHash, r.partitions)
	closestPeers := yacymodel.SeedsClosestToPosition(
		peersEligibleForReplicas(peersAcceptingRemoteIndex(reachablePeers), r.eligibility),
		position,
		r.redundancy,
	)
	responsible := r.responsibleHolders(ctx, holders, position, closestPeers)

	stale := StaleReplicas{
		Posting: postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash},
		Peers:   holdersNoLongerResponsible(holders, responsible),
	}
	missing := MissingReplicas{Posting: posting}
	if len(responsible) >= r.redundancy {
		return missing, stale, nil
	}
	missing.ReplicasNeeded = r.redundancy - len(responsible)
	missing.Seeds = missingSeeds(closestPeers, responsible, missing.ReplicasNeeded)

	return missing, stale, nil
}

func (r *Shortfall) responsibleHolders(
	ctx context.Context,
	holders []yacymodel.Hash,
	position yacymodel.DHTPosition,
	closestPeers []yacymodel.Seed,
) map[yacymodel.Hash]struct{} {
	responsible := make(map[yacymodel.Hash]struct{}, len(holders))
	for _, peer := range holders {
		if r.stillResponsible(ctx, peer, position, closestPeers) {
			responsible[peer] = struct{}{}
		}
	}

	return responsible
}

func holdersNoLongerResponsible(
	holders []yacymodel.Hash,
	responsible map[yacymodel.Hash]struct{},
) []yacymodel.Hash {
	var lost []yacymodel.Hash
	for _, peer := range holders {
		if _, held := responsible[peer]; !held {
			lost = append(lost, peer)
		}
	}

	return lost
}

func missingSeeds(
	closestPeers []yacymodel.Seed,
	responsible map[yacymodel.Hash]struct{},
	replicasNeeded int,
) []yacymodel.Seed {
	missing := make([]yacymodel.Seed, 0, replicasNeeded)
	for _, seed := range closestPeers {
		if len(missing) == replicasNeeded {
			break
		}
		if _, held := responsible[seed.Hash]; !held {
			missing = append(missing, seed)
		}
	}

	return missing
}

func (r *Shortfall) stillResponsible(
	ctx context.Context,
	peer yacymodel.Hash,
	position yacymodel.DHTPosition,
	closestPeers []yacymodel.Seed,
) bool {
	if r.reachability.Reachable(ctx, peer) {
		return r.noFartherThanClosestPeers(peer, position, closestPeers)
	}

	return r.reachability.RecentlyReachable(ctx, peer)
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
