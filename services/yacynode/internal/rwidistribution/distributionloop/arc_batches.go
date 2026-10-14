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
		offers, err := b.offersFrom(ctx, pick, reachablePeers)
		if err == nil && len(offers) > 0 {
			return batchSearch{
				batch:    dueBatch{arc: pick.arc, offers: offers},
				hasBatch: true,
			}, nil
		}
		if err == nil {
			err = b.settleEndOf(ctx, pick)
		}
		if err != nil {
			b.walk.abandon(pick.arc, nil)

			return batchSearch{}, err
		}
	}
}

func (b arcBatches) offersFrom(
	ctx context.Context,
	pick arcPick,
	reachablePeers []yacymodel.Seed,
) ([]wordpartitionoffer.Offer, error) {
	if err := b.schedule.ScheduleAdmissionsWithin(ctx, pick.spans); err != nil {
		return nil, err
	}
	position := pick.position
	for {
		read, err := b.reader.dueEntriesFrom(ctx, pick.spans, position)
		if err != nil {
			return nil, err
		}
		chunk, err := b.chunkOf(ctx, read, reachablePeers)
		if err != nil {
			return nil, err
		}
		b.walk.moveTo(pick.arc, chunk.position)
		if len(chunk.offers) > 0 {
			return chunk.offers, nil
		}
		if read.reachedEnd {
			return nil, nil
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

func (b dueBatch) wordPartitions() []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(b.offers))
	for _, offer := range b.offers {
		wordPartitions = append(wordPartitions, offer.WordPartition())
	}

	return wordPartitions
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
	reserved := b.walk.reserve(read.entries)
	offers, err := b.offers.DueOffersFor(ctx, wordPartitionsOf(reserved), reachablePeers)
	if err != nil {
		b.walk.release(wordPartitionsOf(reserved))

		return readChunk{}, err
	}
	kept := offersWithin(b.postingsBudget, offers)
	b.walk.release(unkeptWordPartitions(reserved, kept))

	return readChunk{
		offers:   kept,
		position: positionAfter(read, len(kept) < len(offers), kept),
	}, nil
}

func wordPartitionsOf(entries []readEntry) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(entries))
	for _, entry := range entries {
		wordPartitions = append(wordPartitions, entry.scheduled.WordPartition)
	}

	return wordPartitions
}

func unkeptWordPartitions(
	reserved []readEntry,
	kept []wordpartitionoffer.Offer,
) []yacymodel.WordPartition {
	var unkept []yacymodel.WordPartition
	for _, entry := range reserved {
		if !slices.ContainsFunc(kept, func(offer wordpartitionoffer.Offer) bool {
			return offer.WordPartition() == entry.scheduled.WordPartition
		}) {
			unkept = append(unkept, entry.scheduled.WordPartition)
		}
	}

	return unkept
}

func positionAfter(
	read arcRead,
	wasTrimmed bool,
	kept []wordpartitionoffer.Offer,
) arcPosition {
	switch {
	case wasTrimmed:
		lastKept := kept[len(kept)-1].WordPartition()
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
