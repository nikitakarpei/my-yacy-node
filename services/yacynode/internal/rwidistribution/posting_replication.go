package rwidistribution

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
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

type postingReplicationReader struct {
	schedule   *postingOfferSchedule
	ledger     *replicaLedger
	postings   rwipostings.PostingIndex
	roster     peerroster.Roster
	partitions yacymodel.DHTRingPartitions
	redundancy int
}

func (r *postingReplicationReader) DueReplication(
	ctx context.Context,
	limit int,
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

		replication, err := r.replicationOf(ctx, posting)
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
) (postingReplication, error) {
	position, err := yacymodel.PostingPosition(posting.WordHash, posting.URLHash, r.partitions)
	if err != nil {
		return postingReplication{}, fmt.Errorf("posting position: %w", err)
	}

	responsible := r.roster.PeersResponsibleFor(ctx, position, r.redundancy)
	responsibleSeeds := make(map[yacymodel.Hash]yacymodel.Seed, len(responsible))
	for _, seed := range responsible {
		responsibleSeeds[seed.Hash] = seed
	}

	replicas, err := r.ledger.Replicas(ctx, posting.WordHash, posting.URLHash)
	if err != nil {
		return postingReplication{}, fmt.Errorf("read replica ledger: %w", err)
	}

	replication := postingReplication{Posting: posting}
	holders := make(map[yacymodel.Hash]struct{}, len(replicas))
	for _, peer := range replicas {
		if _, stillResponsible := responsibleSeeds[peer]; stillResponsible {
			holders[peer] = struct{}{}
		} else {
			replication.PeerHashesNoLongerResponsible = append(
				replication.PeerHashesNoLongerResponsible,
				peer,
			)
		}
	}

	if len(holders) >= r.redundancy {
		return replication, nil
	}
	replication.CopiesNeeded = r.redundancy - len(holders)

	for _, seed := range responsible {
		if _, held := holders[seed.Hash]; !held {
			replication.SeedsMissingCopy = append(replication.SeedsMissingCopy, seed)
		}
	}

	return replication, nil
}
