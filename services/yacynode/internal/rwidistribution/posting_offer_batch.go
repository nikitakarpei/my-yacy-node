package rwidistribution

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const postingsPerPeerCap = 1000

type postingOffer struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type offerPlan struct {
	Offers    []postingOffer
	Satisfied []duePosting
	Stalled   []duePosting
	Drained   int
}

type batchBuilder struct {
	schedule   *offerSchedule
	ledger     *replicaLedger
	postings   rwipostings.PostingIndex
	roster     peerroster.Roster
	observer   OfferObserver
	partitions yacymodel.DHTRingPartitions
	redundancy int
}

func (b *batchBuilder) Build(ctx context.Context, limit int) (offerPlan, error) {
	due, err := b.schedule.DueBatch(ctx, limit)
	if err != nil {
		return offerPlan{}, fmt.Errorf("drain due postings: %w", err)
	}

	plan := offerPlan{Drained: len(due)}
	byPeer := make(map[yacymodel.Hash]*postingOffer)

	for _, entry := range due {
		posting, found, err := b.postings.Posting(ctx, entry.Word, entry.URL)
		if err != nil {
			return offerPlan{}, fmt.Errorf("read posting: %w", err)
		}
		if !found {
			continue
		}

		targeted, err := b.planOne(ctx, entry, posting, byPeer)
		if err != nil {
			return offerPlan{}, err
		}

		switch {
		case targeted == nil:
			plan.Satisfied = append(plan.Satisfied, entry)
		case len(targeted) == 0:
			plan.Stalled = append(plan.Stalled, entry)
		}
	}

	plan.Offers = make([]postingOffer, 0, len(byPeer))
	for _, offer := range byPeer {
		plan.Offers = append(plan.Offers, *offer)
	}

	return plan, nil
}

func (b *batchBuilder) planOne(
	ctx context.Context,
	entry duePosting,
	posting yacymodel.RWIPosting,
	byPeer map[yacymodel.Hash]*postingOffer,
) ([]yacymodel.Hash, error) {
	urlHash, err := yacymodel.ParseURLHash(entry.URL.String())
	if err != nil {
		return nil, fmt.Errorf("parse posting url: %w", err)
	}

	position, err := yacymodel.PostingPosition(entry.Word, urlHash, b.partitions)
	if err != nil {
		return nil, fmt.Errorf("posting position: %w", err)
	}

	responsible := b.roster.PeersResponsibleFor(ctx, position, b.redundancy)
	responsibleSeeds := make(map[yacymodel.Hash]yacymodel.Seed, len(responsible))
	for _, seed := range responsible {
		responsibleSeeds[seed.Hash] = seed
	}

	remaining, dropped, err := b.ledger.Prune(
		ctx,
		entry.Word,
		entry.URL,
		func(peer yacymodel.Hash) bool {
			_, stillResponsible := responsibleSeeds[peer]

			return stillResponsible
		},
	)
	if err != nil {
		return nil, fmt.Errorf("prune replica ledger: %w", err)
	}
	if dropped > 0 {
		b.observer.ObserveLedgerPrune(dropped)
	}

	if len(remaining) >= b.redundancy {
		return nil, nil
	}

	targeted := make([]yacymodel.Hash, 0, len(responsibleSeeds))
	for peer, seed := range responsibleSeeds {
		if containsHash(remaining, peer) {
			continue
		}

		offer := byPeer[peer]
		if offer == nil {
			offer = &postingOffer{Peer: seed}
			byPeer[peer] = offer
		}
		if len(offer.Postings) >= postingsPerPeerCap {
			continue
		}

		offer.Postings = append(offer.Postings, posting)
		targeted = append(targeted, peer)
	}

	return targeted, nil
}

func containsHash(hashes []yacymodel.Hash, target yacymodel.Hash) bool {
	for _, hash := range hashes {
		if hash == target {
			return true
		}
	}

	return false
}
