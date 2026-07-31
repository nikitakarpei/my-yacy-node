package rwidistribution

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type staleReplicas struct {
	Posting duePosting
	Peers   []yacymodel.Hash
}

type postingReplication struct {
	RedundancyMet            bool
	PeersMissingCopy         []yacymodel.Seed
	PeersNoLongerResponsible []yacymodel.Hash
}

type postingOfferPlan struct {
	Offers        []postingOffer
	Replicated    []duePosting
	Unoffered     []duePosting
	Drained       int
	StaleReplicas []staleReplicas
}

type postingOfferPlanner struct {
	schedule          *postingOfferSchedule
	ledger            *replicaLedger
	postings          rwipostings.PostingIndex
	roster            peerroster.Roster
	observer          PostingOfferCycleObserver
	partitions        yacymodel.DHTRingPartitions
	redundancy        int
	minReachablePeers int
}

func (p *postingOfferPlanner) Plan(ctx context.Context, limit int) (postingOfferPlan, error) {
	reachable := len(p.roster.ReachablePeers(ctx))
	if reachable < p.minReachablePeers {
		slog.DebugContext(
			ctx,
			"distribution cycle skipped: too few reachable peers",
			slog.Int("reachablePeers", reachable),
			slog.Int("minReachablePeers", p.minReachablePeers),
		)
		p.observer.ObserveCycleSkipped(reachable)

		return postingOfferPlan{}, nil
	}

	due, err := p.schedule.DueBatch(ctx, limit)
	if err != nil {
		return postingOfferPlan{}, fmt.Errorf("drain due postings: %w", err)
	}

	plan := postingOfferPlan{Drained: len(due)}
	batch := newPostingOfferBatch()

	for _, entry := range due {
		posting, found, err := p.postings.Posting(ctx, entry.Word, entry.URL)
		if err != nil {
			return postingOfferPlan{}, fmt.Errorf("read posting: %w", err)
		}
		if !found {
			continue
		}

		replication, err := p.replicationOf(ctx, entry)
		if err != nil {
			return postingOfferPlan{}, err
		}
		if len(replication.PeersNoLongerResponsible) > 0 {
			plan.StaleReplicas = append(plan.StaleReplicas, staleReplicas{
				Posting: entry,
				Peers:   replication.PeersNoLongerResponsible,
			})
		}

		offered := 0
		for _, peer := range replication.PeersMissingCopy {
			if batch.Add(peer, posting) {
				offered++
			}
		}

		switch {
		case replication.RedundancyMet:
			plan.Replicated = append(plan.Replicated, entry)
		case offered == 0:
			plan.Unoffered = append(plan.Unoffered, entry)
		}
	}

	plan.Offers = batch.Offers()

	return plan, nil
}

func (p *postingOfferPlanner) replicationOf(
	ctx context.Context,
	entry duePosting,
) (postingReplication, error) {
	position, err := yacymodel.PostingPosition(entry.Word, entry.URL, p.partitions)
	if err != nil {
		return postingReplication{}, fmt.Errorf("posting position: %w", err)
	}

	responsible := p.roster.PeersResponsibleFor(ctx, position, p.redundancy)
	responsibleSeeds := make(map[yacymodel.Hash]yacymodel.Seed, len(responsible))
	for _, seed := range responsible {
		responsibleSeeds[seed.Hash] = seed
	}

	replicas, err := p.ledger.Replicas(ctx, entry.Word, entry.URL)
	if err != nil {
		return postingReplication{}, fmt.Errorf("read replica ledger: %w", err)
	}

	var replication postingReplication
	holders := make(map[yacymodel.Hash]struct{}, len(replicas))
	for _, peer := range replicas {
		if _, stillResponsible := responsibleSeeds[peer]; stillResponsible {
			holders[peer] = struct{}{}
		} else {
			replication.PeersNoLongerResponsible = append(
				replication.PeersNoLongerResponsible,
				peer,
			)
		}
	}

	if len(holders) >= p.redundancy {
		replication.RedundancyMet = true

		return replication, nil
	}

	for _, seed := range responsible {
		if _, held := holders[seed.Hash]; !held {
			replication.PeersMissingCopy = append(replication.PeersMissingCopy, seed)
		}
	}

	return replication, nil
}
