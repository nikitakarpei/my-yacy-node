package distributionloop

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type walkedList interface {
	batchAfterLastRead(
		ctx context.Context,
		isInFlight func(yacymodel.WordPartition) bool,
	) (listBatch, bool, error)
}

type listBatch interface {
	wordPartitions() []yacymodel.WordPartition
	offersFor(
		ctx context.Context,
		reachablePeers []yacymodel.Seed,
	) (wordpartitionoffer.Batch, error)
}

type undeliveredPostingsList struct {
	vault           *vault.Vault
	undelivered     *undeliveredpostings.UndeliveredPostings
	offers          *wordpartitionoffer.WordPartitionOffers
	postingsBudget  int
	lastReadPosting undeliveredpostings.UndeliveredPosting
}

func (l *undeliveredPostingsList) batchAfterLastRead(
	ctx context.Context,
	isInFlight func(yacymodel.WordPartition) bool,
) (listBatch, bool, error) {
	var (
		readWordPartitions []undeliveredpostings.UndeliveredWordPartition
		reachedEnd         bool
	)
	if err := l.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		readWordPartitions, reachedEnd, err = l.undelivered.WordPartitionsAfter(
			tx, l.lastReadPosting, l.postingsBudget)

		return err
	}); err != nil {
		return nil, false, err
	}
	l.lastReadPosting = undeliveredpostings.UndeliveredPosting{}
	if !reachedEnd {
		l.lastReadPosting = readWordPartitions[len(readWordPartitions)-1].LastPosting()
	}

	return undeliveredBatch{
		offers:      l.offers,
		undelivered: undeliveredOutside(readWordPartitions, isInFlight),
	}, reachedEnd, nil
}

func undeliveredOutside(
	readWordPartitions []undeliveredpostings.UndeliveredWordPartition,
	isInFlight func(yacymodel.WordPartition) bool,
) []undeliveredpostings.UndeliveredWordPartition {
	var outsideInFlight []undeliveredpostings.UndeliveredWordPartition
	for _, undelivered := range readWordPartitions {
		if !isInFlight(undelivered.WordPartition) {
			outsideInFlight = append(outsideInFlight, undelivered)
		}
	}

	return outsideInFlight
}

type undeliveredBatch struct {
	offers      *wordpartitionoffer.WordPartitionOffers
	undelivered []undeliveredpostings.UndeliveredWordPartition
}

func (b undeliveredBatch) wordPartitions() []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(b.undelivered))
	for _, undelivered := range b.undelivered {
		wordPartitions = append(wordPartitions, undelivered.WordPartition)
	}

	return wordPartitions
}

func (b undeliveredBatch) offersFor(
	ctx context.Context,
	reachablePeers []yacymodel.Seed,
) (wordpartitionoffer.Batch, error) {
	return b.offers.UndeliveredOffersFor(ctx, b.undelivered, reachablePeers)
}

type dueRefreshesList struct {
	vault           *vault.Vault
	holders         *wordpartitionholders.Holders
	offers          *wordpartitionoffer.WordPartitionOffers
	refreshesBudget int
	lastReadRefresh wordpartitionholders.ScheduledRefresh
}

func (l *dueRefreshesList) batchAfterLastRead(
	ctx context.Context,
	isInFlight func(yacymodel.WordPartition) bool,
) (listBatch, bool, error) {
	var (
		readRefreshes []wordpartitionholders.ScheduledRefresh
		reachedEnd    bool
	)
	if err := l.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		readRefreshes, reachedEnd, err = l.holders.DueRefreshesAfter(
			tx, l.lastReadRefresh, l.refreshesBudget)

		return err
	}); err != nil {
		return nil, false, err
	}
	l.lastReadRefresh = wordpartitionholders.ScheduledRefresh{}
	if !reachedEnd {
		l.lastReadRefresh = readRefreshes[len(readRefreshes)-1]
	}

	return refreshBatch{
		offers:    l.offers,
		refreshes: refreshesOutside(readRefreshes, isInFlight),
	}, reachedEnd, nil
}

func refreshesOutside(
	readRefreshes []wordpartitionholders.ScheduledRefresh,
	isInFlight func(yacymodel.WordPartition) bool,
) []yacymodel.WordPartition {
	var outsideInFlight []yacymodel.WordPartition
	for _, refresh := range readRefreshes {
		if !isInFlight(refresh.WordPartition) {
			outsideInFlight = append(outsideInFlight, refresh.WordPartition)
		}
	}

	return outsideInFlight
}

type refreshBatch struct {
	offers    *wordpartitionoffer.WordPartitionOffers
	refreshes []yacymodel.WordPartition
}

func (b refreshBatch) wordPartitions() []yacymodel.WordPartition {
	return b.refreshes
}

func (b refreshBatch) offersFor(
	ctx context.Context,
	reachablePeers []yacymodel.Seed,
) (wordpartitionoffer.Batch, error) {
	return b.offers.RefreshOffersFor(ctx, b.refreshes, reachablePeers)
}
