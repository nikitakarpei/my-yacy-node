// Package wordpartitionoffer decides what each peer gets for the word partitions
// of a batch: a holder the postings it lacks, a missing holder or a due refresh
// the whole word partition. A word partition waits while a recipient is in
// cooldown, and one in retry goes out only as a due refresh. Excluded peers are
// left out of the DHT ring.
package wordpartitionoffer

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
)

type PostingIndex interface {
	PostingOf(
		tx *vault.Txn,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
	PostingsOfWordIn(
		tx *vault.Txn,
		word yacymodel.Hash,
		partition uint,
		visit func(yacymodel.RWIPosting) (bool, error),
	) error
}

type Reachability interface {
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
	IsRecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type PeerEligibility interface {
	IsInCooldown(peer yacymodel.Hash) bool
	IsExcluded(peer yacymodel.Hash) bool
}

type WaitReason string

const (
	WaitRecipientInCooldown  WaitReason = "recipient_in_cooldown"
	WaitWordPartitionInRetry WaitReason = "word_partition_in_retry"
	WaitNoRecipient          WaitReason = "no_recipient"
)

type Config struct {
	Partitions yacymodel.DHTRingPartitions
	Self       yacymodel.Hash
	Redundancy int
}

type WordPartitionOffers struct {
	vault        *vault.Vault
	undelivered  *undeliveredpostings.UndeliveredPostings
	holders      *wordpartitionholders.Holders
	postings     PostingIndex
	reachability Reachability
	eligibility  PeerEligibility
	config       Config
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators and the config
func New(
	v *vault.Vault,
	undelivered *undeliveredpostings.UndeliveredPostings,
	holders *wordpartitionholders.Holders,
	postings PostingIndex,
	reachability Reachability,
	eligibility PeerEligibility,
	config Config,
) *WordPartitionOffers {
	return &WordPartitionOffers{
		vault:        v,
		undelivered:  undelivered,
		holders:      holders,
		postings:     postings,
		reachability: reachability,
		eligibility:  eligibility,
		config:       config,
	}
}

func (o *WordPartitionOffers) UndeliveredOffersFor(
	ctx context.Context,
	undeliveredWordPartitions []undeliveredpostings.UndeliveredWordPartition,
	reachablePeers []yacymodel.Seed,
) (Batch, error) {
	return o.batchFor(
		ctx,
		reachablePeers,
		func(tx *vault.Txn, acceptingPeers yacymodel.SeedRing, batch *Batch) error {
			return o.addUndeliveredWordPartitions(
				ctx, tx, undeliveredWordPartitions, acceptingPeers, batch)
		},
	)
}

func (o *WordPartitionOffers) RefreshOffersFor(
	ctx context.Context,
	dueWordPartitions []yacymodel.WordPartition,
	reachablePeers []yacymodel.Seed,
) (Batch, error) {
	return o.batchFor(
		ctx,
		reachablePeers,
		func(tx *vault.Txn, acceptingPeers yacymodel.SeedRing, batch *Batch) error {
			return o.addDueRefreshes(ctx, tx, dueWordPartitions, acceptingPeers, batch)
		},
	)
}

func (o *WordPartitionOffers) batchFor(
	ctx context.Context,
	reachablePeers []yacymodel.Seed,
	addTo func(tx *vault.Txn, acceptingPeers yacymodel.SeedRing, batch *Batch) error,
) (Batch, error) {
	acceptingPeers := o.acceptingPeersAmong(reachablePeers)
	var batch Batch
	if err := o.vault.View(ctx, func(tx *vault.Txn) error {
		return addTo(tx, acceptingPeers, &batch)
	}); err != nil {
		return Batch{}, err
	}

	return batch, nil
}

func (o *WordPartitionOffers) acceptingPeersAmong(
	reachablePeers []yacymodel.Seed,
) yacymodel.SeedRing {
	acceptingPeers := make([]yacymodel.Seed, 0, len(reachablePeers))
	for _, seed := range reachablePeers {
		if seed.AcceptsRemoteIndex() && !o.eligibility.IsExcluded(seed.Hash) {
			acceptingPeers = append(acceptingPeers, seed)
		}
	}

	return yacymodel.SeedRingFrom(acceptingPeers)
}

func (o *WordPartitionOffers) addUndeliveredWordPartitions(
	ctx context.Context,
	tx *vault.Txn,
	undeliveredWordPartitions []undeliveredpostings.UndeliveredWordPartition,
	acceptingPeers yacymodel.SeedRing,
	batch *Batch,
) error {
	for _, undelivered := range undeliveredWordPartitions {
		isInRetry, err := o.holders.IsInRetry(tx, undelivered.WordPartition)
		if err != nil {
			return err
		}
		if isInRetry {
			batch.addWait(WaitWordPartitionInRetry)

			continue
		}
		peers, err := o.peersOf(ctx, tx, undelivered.WordPartition, acceptingPeers)
		if err != nil {
			return err
		}
		if waitReason, mustWait := peers.waitReasonFrom(o.eligibility); mustWait {
			batch.addWait(waitReason)

			continue
		}
		offer, err := o.undeliveredOfferFor(tx, undelivered.Postings, peers)
		if err != nil {
			return err
		}
		batch.addOffer(offer)
	}

	return nil
}

func (o *WordPartitionOffers) peersOf(
	ctx context.Context,
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	acceptingPeers yacymodel.SeedRing,
) (wordPartitionPeers, error) {
	recordedHolders, err := o.holders.Of(tx, wordPartition)
	if err != nil {
		return wordPartitionPeers{}, err
	}
	responsiblePeers := responsiblePeersFor(
		yacymodel.DHTRingPositionOfWordPartition(wordPartition, o.config.Partitions),
		acceptingPeers,
		o.config.Self,
		o.config.Redundancy,
	)

	return wordPartitionPeersFrom(
		wordPartition,
		responsiblePeers,
		acceptingPeers,
		recordedHolders,
		o.holdersAmong(ctx, recordedHolders, responsiblePeers),
	), nil
}

func (o *WordPartitionOffers) holdersAmong(
	ctx context.Context,
	recordedHolders []yacymodel.Hash,
	responsiblePeers responsiblePeers,
) []yacymodel.Hash {
	var holdersAmongResponsiblePeers []yacymodel.Hash
	for _, holder := range recordedHolders {
		if o.countsAmong(ctx, holder, responsiblePeers) {
			holdersAmongResponsiblePeers = append(holdersAmongResponsiblePeers, holder)
		}
	}

	return holdersAmongResponsiblePeers
}

func (o *WordPartitionOffers) countsAmong(
	ctx context.Context,
	holder yacymodel.Hash,
	responsiblePeers responsiblePeers,
) bool {
	if o.eligibility.IsExcluded(holder) {
		return false
	}
	if o.reachability.IsReachable(ctx, holder) {
		return responsiblePeers.contains(holder)
	}

	return o.reachability.IsRecentlyReachable(ctx, holder)
}

func (o *WordPartitionOffers) undeliveredOfferFor(
	tx *vault.Txn,
	undelivered []undeliveredpostings.UndeliveredPosting,
	peers wordPartitionPeers,
) (WordPartitionOffer, error) {
	var undeliveredValues []yacymodel.RWIPosting
	if peers.hasHoldingPeers() {
		var err error
		if undeliveredValues, err = o.storedPostingsOf(tx, undelivered); err != nil {
			return WordPartitionOffer{}, err
		}
	}
	var wholeWordPartition []yacymodel.RWIPosting
	if peers.hasMissingHolders() {
		var err error
		if wholeWordPartition, err = o.postingsIn(tx, peers.wordPartition); err != nil {
			return WordPartitionOffer{}, err
		}
	}

	return peers.undeliveredOfferWith(
		undelivered,
		undeliveredValues,
		wholeWordPartition,
	), nil
}

func (o *WordPartitionOffers) storedPostingsOf(
	tx *vault.Txn,
	undelivered []undeliveredpostings.UndeliveredPosting,
) ([]yacymodel.RWIPosting, error) {
	storedPostings := make([]yacymodel.RWIPosting, 0, len(undelivered))
	for _, posting := range undelivered {
		stored, found, err := o.postings.PostingOf(tx, posting.WordPartition.Word, posting.URL)
		if err != nil {
			return nil, err
		}
		if found {
			storedPostings = append(storedPostings, stored)
		}
	}

	return storedPostings, nil
}

func (o *WordPartitionOffers) postingsIn(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) ([]yacymodel.RWIPosting, error) {
	var wholeWordPartition []yacymodel.RWIPosting
	err := o.postings.PostingsOfWordIn(
		tx,
		wordPartition.Word,
		wordPartition.Partition,
		func(posting yacymodel.RWIPosting) (bool, error) {
			wholeWordPartition = append(wholeWordPartition, posting)

			return true, nil
		},
	)

	return wholeWordPartition, err
}

func (o *WordPartitionOffers) addDueRefreshes(
	ctx context.Context,
	tx *vault.Txn,
	dueWordPartitions []yacymodel.WordPartition,
	acceptingPeers yacymodel.SeedRing,
	batch *Batch,
) error {
	for _, wordPartition := range dueWordPartitions {
		peers, err := o.peersOf(ctx, tx, wordPartition, acceptingPeers)
		if err != nil {
			return err
		}
		if waitReason, mustWait := peers.waitReasonFrom(o.eligibility); mustWait {
			batch.addWait(waitReason)

			continue
		}
		offer, err := o.refreshOfferFor(tx, peers)
		if err != nil {
			return err
		}
		batch.addOffer(offer)
	}

	return nil
}

func (o *WordPartitionOffers) refreshOfferFor(
	tx *vault.Txn,
	peers wordPartitionPeers,
) (WordPartitionOffer, error) {
	undelivered, err := o.undelivered.PostingsOf(tx, peers.wordPartition)
	if err != nil {
		return WordPartitionOffer{}, err
	}
	wholeWordPartition, err := o.postingsIn(tx, peers.wordPartition)
	if err != nil {
		return WordPartitionOffer{}, err
	}

	return peers.refreshOfferWith(undelivered, wholeWordPartition), nil
}
