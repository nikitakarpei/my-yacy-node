// Package wordpartitionoffer decides, one ring sector at a time, which word
// partitions to offer now and what each peer gets. A holder gets the postings
// it does not have yet, a missing holder gets the whole word partition, and a
// due refresh sends the whole word partition again. A word partition waits
// while it has no recipient or one of its recipients is in cooldown.
package wordpartitionoffer

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
)

type Cursor struct {
	lastVisited undeliveredpostings.UndeliveredPosting
}

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

type Cooldowns interface {
	CooldownLeftOf(peer yacymodel.Hash) time.Duration
}

type Observer interface {
	ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int)
}

type Config struct {
	Partitions       yacymodel.DHTRingPartitions
	Self             yacymodel.Hash
	Redundancy       int
	PostingsPerBatch int
}

type WordPartitionOffers struct {
	vault        *vault.Vault
	undelivered  *undeliveredpostings.UndeliveredPostings
	holders      *wordpartitionholders.Holders
	postings     PostingIndex
	reachability Reachability
	cooldowns    Cooldowns
	observer     Observer
	config       Config
}

//nolint:revive // argument-limit: seven explicit, independently-meaningful collaborators and the config
func New(
	v *vault.Vault,
	undelivered *undeliveredpostings.UndeliveredPostings,
	holders *wordpartitionholders.Holders,
	postings PostingIndex,
	reachability Reachability,
	cooldowns Cooldowns,
	observer Observer,
	config Config,
) *WordPartitionOffers {
	return &WordPartitionOffers{
		vault:        v,
		undelivered:  undelivered,
		holders:      holders,
		postings:     postings,
		reachability: reachability,
		cooldowns:    cooldowns,
		observer:     observer,
		config:       config,
	}
}

func (o *WordPartitionOffers) ReportPeersAcceptingRemoteIndex(reachablePeers []yacymodel.Seed) {
	o.observer.ObservePeersAcceptingRemoteIndexPerDHTRingSector(
		yacymodel.SeedsPerDHTRingSector(peersAcceptingRemoteIndex(reachablePeers)),
	)
}

func peersAcceptingRemoteIndex(peers []yacymodel.Seed) []yacymodel.Seed {
	acceptingPeers := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if acceptsRemoteIndex(seed) {
			acceptingPeers = append(acceptingPeers, seed)
		}
	}

	return acceptingPeers
}

func acceptsRemoteIndex(seed yacymodel.Seed) bool {
	capabilities, capabilitiesKnown := seed.Capabilities.Get()

	return capabilitiesKnown && capabilities.AcceptRemoteIndex
}

func (o *WordPartitionOffers) DueNowIn(
	ctx context.Context,
	sector yacymodel.DHTRingSector,
	from Cursor,
	reachablePeers []yacymodel.Seed,
) (offers []WordPartitionOffer, next Cursor, err error) {
	acceptingPeers := yacymodel.SeedRingFrom(peersAcceptingRemoteIndex(reachablePeers))
	batch := offerBatchFrom(from, o.config.PostingsPerBatch)
	if err := o.vault.View(ctx, func(tx *vault.Txn) error {
		if err := o.addUndeliveredWordPartitions(
			ctx,
			tx,
			sector,
			acceptingPeers,
			&batch,
		); err != nil {
			return err
		}

		return o.addDueRefreshes(ctx, tx, sector, acceptingPeers, &batch)
	}); err != nil {
		return nil, Cursor{}, err
	}

	return batch.offers, batch.next, nil
}

func (o *WordPartitionOffers) addUndeliveredWordPartitions(
	ctx context.Context,
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
	acceptingPeers yacymodel.SeedRing,
	batch *offerBatch,
) error {
	for !batch.isFull() {
		undelivered, err := o.nextUndeliveredWordPartition(tx, sector, batch.next)
		if err != nil {
			return err
		}
		if len(undelivered) == 0 {
			batch.reachedSectorEnd()

			return nil
		}
		batch.walkedPast(undelivered)
		peers, err := o.peersOf(ctx, tx, undelivered[0].WordPartition, acceptingPeers)
		if err != nil {
			return err
		}
		if peers.mustWaitFor(o.cooldowns) {
			continue
		}
		offer, err := o.undeliveredOfferFor(tx, undelivered, peers)
		if err != nil {
			return err
		}
		batch.add(offer)
	}

	return nil
}

func (o *WordPartitionOffers) nextUndeliveredWordPartition(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
	from Cursor,
) ([]undeliveredpostings.UndeliveredPosting, error) {
	var undelivered []undeliveredpostings.UndeliveredPosting
	err := o.undelivered.PostingsIn(
		tx,
		sector,
		from.lastVisited,
		func(posting undeliveredpostings.UndeliveredPosting) (bool, error) {
			if len(undelivered) > 0 && posting.WordPartition != undelivered[0].WordPartition {
				return false, nil
			}
			undelivered = append(undelivered, posting)

			return true, nil
		},
	)

	return undelivered, err
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
	undeliveredValues, err := o.storedPostingsFor(tx, peers.holdingPeers, undelivered)
	if err != nil {
		return WordPartitionOffer{}, err
	}
	wholeWordPartition, err := o.wholeWordPartitionFor(
		tx,
		peers.missingHolders,
		peers.wordPartition,
	)
	if err != nil {
		return WordPartitionOffer{}, err
	}

	return peers.undeliveredOfferWith(
		undelivered,
		undeliveredValues,
		wholeWordPartition,
	), nil
}

func (o *WordPartitionOffers) storedPostingsFor(
	tx *vault.Txn,
	recipients []yacymodel.Seed,
	undelivered []undeliveredpostings.UndeliveredPosting,
) ([]yacymodel.RWIPosting, error) {
	if len(recipients) == 0 {
		return nil, nil
	}
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

func (o *WordPartitionOffers) wholeWordPartitionFor(
	tx *vault.Txn,
	recipients []yacymodel.Seed,
	wordPartition yacymodel.WordPartition,
) ([]yacymodel.RWIPosting, error) {
	if len(recipients) == 0 {
		return nil, nil
	}
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
	sector yacymodel.DHTRingSector,
	acceptingPeers yacymodel.SeedRing,
	batch *offerBatch,
) error {
	if batch.isFull() {
		return nil
	}
	dueWordPartitions, err := o.holders.DueRefreshesIn(tx, sector, batch.roomLeft())
	if err != nil {
		return err
	}
	for _, wordPartition := range dueWordPartitions {
		if batch.isFull() {
			return nil
		}
		if batch.holds(wordPartition) {
			continue
		}
		peers, err := o.peersOf(ctx, tx, wordPartition, acceptingPeers)
		if err != nil {
			return err
		}
		if peers.mustWaitFor(o.cooldowns) {
			continue
		}
		offer, err := o.refreshOfferFor(tx, peers)
		if err != nil {
			return err
		}
		batch.add(offer)
	}

	return nil
}

func (o *WordPartitionOffers) refreshOfferFor(
	tx *vault.Txn,
	peers wordPartitionPeers,
) (WordPartitionOffer, error) {
	undelivered, err := o.undeliveredPostingsOf(tx, peers.wordPartition)
	if err != nil {
		return WordPartitionOffer{}, err
	}
	wholeWordPartition, err := o.wholeWordPartitionFor(tx, peers.recipients(), peers.wordPartition)
	if err != nil {
		return WordPartitionOffer{}, err
	}

	return peers.refreshOfferWith(undelivered, wholeWordPartition), nil
}

func (o *WordPartitionOffers) undeliveredPostingsOf(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) ([]undeliveredpostings.UndeliveredPosting, error) {
	var undelivered []undeliveredpostings.UndeliveredPosting
	err := o.undelivered.PostingsOf(
		tx,
		wordPartition,
		func(posting undeliveredpostings.UndeliveredPosting) (bool, error) {
			undelivered = append(undelivered, posting)

			return true, nil
		},
	)

	return undelivered, err
}
