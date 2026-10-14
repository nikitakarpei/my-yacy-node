// Package wordpartitionhandoff decides which word partitions this node may stop
// holding, and deletes them. A word partition may go once none of its postings
// is undelivered and the redundancy in its responsible peers, strictly closer to
// its DHT position than this node, hold it. A node that keeps every word
// partition hands off none.
package wordpartitionhandoff

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Config struct {
	Partitions yacymodel.DHTRingPartitions
	Self       yacymodel.Hash
	Redundancy int
}

type Handoff struct {
	purger      rwipostings.PostingPurger
	undelivered *undeliveredpostings.Postings
	schedule    *wordpartitionschedule.Schedule
	config      Config
}

func New(
	purger rwipostings.PostingPurger,
	undelivered *undeliveredpostings.Postings,
	schedule *wordpartitionschedule.Schedule,
	config Config,
) *Handoff {
	return &Handoff{
		purger:      purger,
		undelivered: undelivered,
		schedule:    schedule,
		config:      config,
	}
}

func (h *Handoff) HandOffIfHeldByResponsiblePeers(
	ctx context.Context,
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	currentHolders []yacymodel.Hash,
	responsiblePeers []yacymodel.Hash,
) (amountOfPostingsHandedOff int, err error) {
	if !h.isHeldByResponsiblePeers(wordPartition, currentHolders, responsiblePeers) {
		return 0, nil
	}
	hasUndeliveredPostings, err := h.undelivered.AnyIn(tx, wordPartition)
	if err != nil || hasUndeliveredPostings {
		return 0, err
	}
	amountOfPostingsHandedOff, err = h.purger.PurgePostingsOfWordIn(
		tx,
		wordPartition.Word,
		wordPartition.Partition,
	)
	if err != nil {
		return 0, err
	}
	if err := h.schedule.Forget(tx, wordPartition); err != nil {
		return 0, err
	}
	slog.DebugContext(ctx, "word partition handed off to responsible peers",
		slog.String("word", wordPartition.Word.String()),
		slog.Uint64("partition", uint64(wordPartition.Partition)),
		slog.Int("postings", amountOfPostingsHandedOff))

	return amountOfPostingsHandedOff, nil
}

func (h *Handoff) isHeldByResponsiblePeers(
	wordPartition yacymodel.WordPartition,
	currentHolders []yacymodel.Hash,
	responsiblePeers []yacymodel.Hash,
) bool {
	position := yacymodel.DHTRingPositionOfWordPartition(wordPartition, h.config.Partitions)
	amountOfCloserResponsibleHolders := 0
	for _, peer := range currentHolders {
		if slices.Contains(responsiblePeers, peer) &&
			yacymodel.CloserToDHTRingPosition(peer, h.config.Self, position) {
			amountOfCloserResponsibleHolders++
		}
	}

	return amountOfCloserResponsibleHolders >= h.config.Redundancy
}
