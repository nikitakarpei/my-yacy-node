package rwidistribution

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type postingReplication struct {
	Posting                       yacymodel.RWIPosting
	CopiesNeeded                  int
	SeedsMissingCopy              []yacymodel.Seed
	PeerHashesNoLongerResponsible []yacymodel.Hash
}

type dueReplication struct {
	Postings []postingReplication
	Gone     []postingIdentity
}

type peerReachability interface {
	Reachable(ctx context.Context, peer yacymodel.Hash) bool
	RecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type postingReplicationReader struct {
	schedule     *postingOfferSchedule
	ledger       *replicaLedger
	postings     rwipostings.PostingIndex
	reachability peerReachability
	partitions   yacymodel.DHTRingPartitions
	redundancy   int
}

func (r *postingReplicationReader) DueReplication(
	ctx context.Context,
	limit int,
	reachablePeers []yacymodel.Seed,
) (dueReplication, error) {
	due, err := r.schedule.DuePostings(ctx, limit)
	if err != nil {
		return dueReplication{}, fmt.Errorf("read due postings: %w", err)
	}

	var result dueReplication
	for _, identity := range due {
		posting, found, err := r.postings.Posting(ctx, identity.Word, identity.URL)
		if err != nil {
			return dueReplication{}, fmt.Errorf("read posting: %w", err)
		}
		if !found {
			result.Gone = append(result.Gone, identity)

			continue
		}

		replication, err := r.replicationOf(ctx, posting, reachablePeers)
		if err != nil {
			return dueReplication{}, err
		}
		result.Postings = append(result.Postings, replication)
	}

	return result, nil
}

func (r *postingReplicationReader) replicationOf(
	ctx context.Context,
	posting yacymodel.RWIPosting,
	reachablePeers []yacymodel.Seed,
) (postingReplication, error) {
	replicas, err := r.ledger.Replicas(ctx, posting.WordHash, posting.URLHash)
	if err != nil {
		return postingReplication{}, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.PostingPosition(posting.WordHash, posting.URLHash, r.partitions)
	closestPeers := yacymodel.SeedsClosestToPosition(
		peersAcceptingRemoteIndex(reachablePeers), position, r.redundancy,
	)
	holders := r.currentHolders(ctx, replicas, position, closestPeers)

	replication := postingReplication{
		Posting:                       posting,
		PeerHashesNoLongerResponsible: peersNoLongerResponsible(replicas, holders),
	}
	if len(holders) >= r.redundancy {
		return replication, nil
	}
	replication.CopiesNeeded = r.redundancy - len(holders)
	replication.SeedsMissingCopy = seedsMissingCopy(closestPeers, holders, replication.CopiesNeeded)

	return replication, nil
}

func (r *postingReplicationReader) currentHolders(
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

func peersNoLongerResponsible(
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

func seedsMissingCopy(
	closestPeers []yacymodel.Seed,
	holders map[yacymodel.Hash]struct{},
	copiesNeeded int,
) []yacymodel.Seed {
	missing := make([]yacymodel.Seed, 0, copiesNeeded)
	for _, seed := range closestPeers {
		if len(missing) == copiesNeeded {
			break
		}
		if _, held := holders[seed.Hash]; !held {
			missing = append(missing, seed)
		}
	}

	return missing
}

func (r *postingReplicationReader) stillResponsible(
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

func (r *postingReplicationReader) noFartherThanClosestPeers(
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
