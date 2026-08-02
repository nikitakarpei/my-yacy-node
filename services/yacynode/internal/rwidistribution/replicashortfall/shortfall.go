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

type Shortfall struct {
	schedule     *postingschedule.Schedule
	replicas     *postingreplicas.Replicas
	postings     rwipostings.PostingIndex
	reachability Reachability
	partitions   yacymodel.DHTRingPartitions
	redundancy   int
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func New(
	schedule *postingschedule.Schedule,
	replicas *postingreplicas.Replicas,
	postings rwipostings.PostingIndex,
	reachability Reachability,
	partitions yacymodel.DHTRingPartitions,
	redundancy int,
) *Shortfall {
	return &Shortfall{
		schedule:     schedule,
		replicas:     replicas,
		postings:     postings,
		reachability: reachability,
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
	replicas, err := r.replicas.Replicas(ctx, posting.WordHash, posting.URLHash)
	if err != nil {
		return MissingReplicas{}, StaleReplicas{}, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.PostingPosition(posting.WordHash, posting.URLHash, r.partitions)
	closestPeers := yacymodel.SeedsClosestToPosition(
		peersAcceptingRemoteIndex(reachablePeers), position, r.redundancy,
	)
	holders := r.currentHolders(ctx, replicas, position, closestPeers)

	stale := StaleReplicas{
		Posting: postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash},
		Peers:   stalePeers(replicas, holders),
	}
	missing := MissingReplicas{Posting: posting}
	if len(holders) >= r.redundancy {
		return missing, stale, nil
	}
	missing.ReplicasNeeded = r.redundancy - len(holders)
	missing.Seeds = missingSeeds(closestPeers, holders, missing.ReplicasNeeded)

	return missing, stale, nil
}

func (r *Shortfall) currentHolders(
	ctx context.Context,
	replicas []yacymodel.Hash,
	position yacymodel.DHTPosition,
	closestPeers []yacymodel.Seed,
) map[yacymodel.Hash]struct{} {
	holders := make(map[yacymodel.Hash]struct{}, len(replicas))
	for _, peer := range replicas {
		if r.stillResponsible(ctx, peer, position, closestPeers) {
			holders[peer] = struct{}{}
		}
	}

	return holders
}

func stalePeers(
	replicas []yacymodel.Hash,
	holders map[yacymodel.Hash]struct{},
) []yacymodel.Hash {
	var lost []yacymodel.Hash
	for _, peer := range replicas {
		if _, held := holders[peer]; !held {
			lost = append(lost, peer)
		}
	}

	return lost
}

func missingSeeds(
	closestPeers []yacymodel.Seed,
	holders map[yacymodel.Hash]struct{},
	replicasNeeded int,
) []yacymodel.Seed {
	missing := make([]yacymodel.Seed, 0, replicasNeeded)
	for _, seed := range closestPeers {
		if len(missing) == replicasNeeded {
			break
		}
		if _, held := holders[seed.Hash]; !held {
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
