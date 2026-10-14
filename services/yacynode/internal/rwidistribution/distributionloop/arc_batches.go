package distributionloop

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
)

type dueBatch struct {
	arc    *arc
	offers []wordpartitionoffer.Offer
}

type batchSearch struct {
	batch    dueBatch
	hasBatch bool
	alarm    wakeAlarm
}

type arcBatches struct {
	walk           *arcWalk
	schedule       *wordpartitionschedule.Schedule
	reader         arcReader
	offers         *wordpartitionoffer.Offers
	postingsBudget int
}

func (b arcBatches) nextFrom(
	ctx context.Context,
	reachablePeers []yacymodel.Seed,
) (batchSearch, error) {
	b.walk.followRing(b.offers.ArcsAmong(reachablePeers))
	for {
		pick, found, alarm := b.walk.pickOpenArc()
		if !found {
			return batchSearch{alarm: alarm}, nil
		}
		offersOfArc, err := b.offersFrom(ctx, pick, reachablePeers)
		switch {
		case err == nil && len(offersOfArc.offers) > 0:
			return batchSearch{
				batch:    dueBatch{arc: pick.arc, offers: offersOfArc.offers},
				hasBatch: true,
			}, nil
		case err == nil && offersOfArc.isWindowClosed:
			b.walk.putBack(pick.arc)

			continue
		case err == nil:
			err = b.settleEndOf(ctx, pick)
		}
		if err != nil {
			b.walk.abandon(pick.arc)

			return batchSearch{}, err
		}
	}
}

type arcOffers struct {
	offers         []wordpartitionoffer.Offer
	isWindowClosed bool
}

func (b arcBatches) offersFrom(
	ctx context.Context,
	pick arcPick,
	reachablePeers []yacymodel.Seed,
) (arcOffers, error) {
	if err := b.schedule.ScheduleAdmissionsWithin(ctx, pick.spans); err != nil {
		return arcOffers{}, err
	}
	position := pick.position
	for {
		if !b.walk.hasFreeWindow(pick.arc) {
			return arcOffers{isWindowClosed: true}, nil
		}
		read, err := b.reader.dueEntriesFrom(ctx, pick.spans, position)
		if err != nil {
			return arcOffers{}, err
		}
		chunk, err := b.chunkOf(ctx, read, reachablePeers)
		if err != nil {
			return arcOffers{}, err
		}
		b.walk.moveTo(pick.arc, chunk.position)
		if len(chunk.offers) > 0 || read.reachedEnd {
			return arcOffers{offers: chunk.offers}, nil
		}
		position = chunk.position
	}
}

func (b arcBatches) settleEndOf(ctx context.Context, pick arcPick) error {
	earliestDueAt, isAnyDue, err := b.reader.earliestDueWithin(ctx, pick.spans)
	if err != nil {
		return err
	}
	b.walk.settleReadEnd(pick, earliestDueAt, isAnyDue)

	return nil
}

type readChunk struct {
	offers   []wordpartitionoffer.Offer
	position arcPosition
}

func (b arcBatches) chunkOf(
	ctx context.Context,
	read arcRead,
	reachablePeers []yacymodel.Seed,
) (readChunk, error) {
	offers, isBudgetSpent, err := b.offers.DueOffersWithin(
		ctx, wordPartitionsOf(read.entries), reachablePeers, b.postingsBudget)
	if err != nil {
		return readChunk{}, err
	}

	return readChunk{
		offers:   offers,
		position: positionAfter(read, isBudgetSpent, offers),
	}, nil
}

func wordPartitionsOf(entries []readEntry) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(entries))
	for _, entry := range entries {
		wordPartitions = append(wordPartitions, entry.scheduled.WordPartition)
	}

	return wordPartitions
}

func positionAfter(
	read arcRead,
	isBudgetSpent bool,
	offers []wordpartitionoffer.Offer,
) arcPosition {
	switch {
	case isBudgetSpent:
		lastKept := offers[len(offers)-1].WordPartition()
		index := slices.IndexFunc(read.entries, func(entry readEntry) bool {
			return entry.scheduled.WordPartition == lastKept
		})

		return positionOf(read.entries[index])
	case read.reachedEnd:
		return arcPosition{}
	}

	return positionOf(read.entries[len(read.entries)-1])
}

func positionOf(entry readEntry) arcPosition {
	return arcPosition{spanIndex: entry.spanIndex, cursor: entry.scheduled}
}
