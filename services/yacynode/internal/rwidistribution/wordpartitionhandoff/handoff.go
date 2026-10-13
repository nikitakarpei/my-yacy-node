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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type PostingIndex interface {
	PostingsOfWordIn(
		tx *vault.Txn,
		word yacymodel.Hash,
		partition uint,
		visit func(yacymodel.RWIPosting) (bool, error),
	) error
}

type Config struct {
	Partitions yacymodel.DHTRingPartitions
	Self       yacymodel.Hash
	Redundancy int
}

type Handoff struct {
	postings    PostingIndex
	purger      rwipostings.PostingPurger
	undelivered *undeliveredpostings.Postings
	holders     *wordpartitionholders.Holders
	config      Config
}

func New(
	postings PostingIndex,
	purger rwipostings.PostingPurger,
	undelivered *undeliveredpostings.Postings,
	holders *wordpartitionholders.Holders,
	config Config,
) *Handoff {
	return &Handoff{
		postings:    postings,
		purger:      purger,
		undelivered: undelivered,
		holders:     holders,
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
	heldPostings, err := h.postingsOf(tx, wordPartition)
	if err != nil {
		return 0, err
	}
	amountOfPostingsHandedOff, err = h.purge(tx, heldPostings)
	if err != nil {
		return 0, err
	}
	if err := h.holders.Forget(tx, wordPartition); err != nil {
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

func (h *Handoff) postingsOf(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) ([]yacymodel.RWIPosting, error) {
	var heldPostings []yacymodel.RWIPosting
	err := h.postings.PostingsOfWordIn(
		tx,
		wordPartition.Word,
		wordPartition.Partition,
		func(posting yacymodel.RWIPosting) (bool, error) {
			heldPostings = append(heldPostings, posting)

			return true, nil
		},
	)

	return heldPostings, err
}

func (h *Handoff) purge(
	tx *vault.Txn,
	heldPostings []yacymodel.RWIPosting,
) (amountOfPurgedPostings int, err error) {
	for _, posting := range heldPostings {
		wasPurged, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash)
		if err != nil {
			return 0, err
		}
		if wasPurged {
			amountOfPurgedPostings++
		}
	}

	return amountOfPurgedPostings, nil
}
