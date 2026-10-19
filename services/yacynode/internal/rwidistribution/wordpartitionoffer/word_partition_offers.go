// Package wordpartitionoffer decides what each peer gets for the due word
// partitions of a batch: a holder the postings it lacks, a missing holder or a
// due refresh the whole word partition. A paused or busy recipient holds its
// word partition back, excluded peers stay off the DHT ring, and a batch stops
// before a peer's budget overflows. An accepted posting is delivered if stored as sent.
package wordpartitionoffer

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
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
	IsPaused(peer yacymodel.Hash) bool
	IsExcluded(peer yacymodel.Hash) bool
	IsBusy(peer yacymodel.Hash) bool
}

type Config struct {
	Partitions yacymodel.DHTRingPartitions
	Self       yacymodel.Hash
	Redundancy int
}

type Offers struct {
	vault        *vault.Vault
	undelivered  *undeliveredpostings.Postings
	schedule     *wordpartitionschedule.Schedule
	postings     PostingIndex
	reachability Reachability
	eligibility  PeerEligibility
	config       Config
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators and the config
func New(
	v *vault.Vault,
	undelivered *undeliveredpostings.Postings,
	schedule *wordpartitionschedule.Schedule,
	postings PostingIndex,
	reachability Reachability,
	eligibility PeerEligibility,
	config Config,
) *Offers {
	return &Offers{
		vault:        v,
		undelivered:  undelivered,
		schedule:     schedule,
		postings:     postings,
		reachability: reachability,
		eligibility:  eligibility,
		config:       config,
	}
}

func (o *Offers) DueOffersWithin(
	ctx context.Context,
	dueWordPartitions []yacymodel.WordPartition,
	reachablePeers []yacymodel.Seed,
	amountOfPostingsPerPeer int,
) (offers []Offer, isBudgetSpent bool, err error) {
	acceptingPeers := o.acceptingPeersAmong(reachablePeers)
	budget := postingsBudgetOf(amountOfPostingsPerPeer)
	if err := o.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		offers, isBudgetSpent, err = o.offersWithin(
			ctx, tx, dueWordPartitions, acceptingPeers, budget)

		return err
	}); err != nil {
		return nil, false, err
	}

	return offers, isBudgetSpent, nil
}

func (o *Offers) offersWithin(
	ctx context.Context,
	tx *vault.Txn,
	dueWordPartitions []yacymodel.WordPartition,
	acceptingPeers yacymodel.SeedRing,
	budget *postingsBudget,
) (offers []Offer, isBudgetSpent bool, err error) {
	for _, wordPartition := range dueWordPartitions {
		peers, err := o.peersOf(ctx, tx, wordPartition, acceptingPeers)
		if err != nil {
			return nil, false, err
		}
		if peers.mustWaitFor(o.eligibility) {
			continue
		}
		offer, err := o.dueOfferFor(tx, peers)
		if err != nil {
			return nil, false, err
		}
		if len(offers) > 0 && budget.wouldOverflowWith(offer) {
			return offers, true, nil
		}
		budget.take(offer)
		offers = append(offers, offer)
	}

	return offers, false, nil
}

func (o *Offers) dueOfferFor(tx *vault.Txn, peers wordPartitionPeers) (Offer, error) {
	undelivered, err := o.undelivered.PostingsOf(tx, peers.wordPartition)
	if err != nil {
		return Offer{}, err
	}
	isRefreshDue, err := o.schedule.IsRefreshDue(tx, peers.wordPartition)
	if err != nil {
		return Offer{}, err
	}
	if len(undelivered) > 0 && !isRefreshDue {
		return o.undeliveredOfferFor(tx, undelivered, peers)
	}

	return o.refreshOfferFor(tx, undelivered, peers)
}

func (o *Offers) ArcsAmong(reachablePeers []yacymodel.Seed) []yacymodel.SeedArc {
	return o.acceptingPeersAmong(reachablePeers).Arcs(o.config.Redundancy)
}

func (o *Offers) acceptingPeersAmong(
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

func (o *Offers) peersOf(
	ctx context.Context,
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	acceptingPeers yacymodel.SeedRing,
) (wordPartitionPeers, error) {
	recordedHolders, err := o.schedule.HoldersOf(tx, wordPartition)
	if err != nil {
		return wordPartitionPeers{}, err
	}
	responsiblePeers := responsiblePeersFor(
		o.positionOf(wordPartition),
		acceptingPeers,
		o.config.Self,
		o.config.Redundancy,
	)

	return wordPartitionPeersFrom(
		wordPartition,
		responsiblePeers,
		acceptingPeers,
		recordedHolders,
		o.keptHoldersOf(ctx, recordedHolders, responsiblePeers),
	), nil
}

func (o *Offers) positionOf(
	wordPartition yacymodel.WordPartition,
) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordPartition(wordPartition, o.config.Partitions)
}

func (o *Offers) keptHoldersOf(
	ctx context.Context,
	recordedHolders []yacymodel.Hash,
	responsiblePeers responsiblePeers,
) []yacymodel.Hash {
	var keptHolders []yacymodel.Hash
	for _, holder := range recordedHolders {
		if o.isKept(ctx, holder, responsiblePeers) {
			keptHolders = append(keptHolders, holder)
		}
	}

	return keptHolders
}

func (o *Offers) isKept(
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

func (o *Offers) undeliveredOfferFor(
	tx *vault.Txn,
	undelivered []undeliveredpostings.Posting,
	peers wordPartitionPeers,
) (Offer, error) {
	var undeliveredValues []yacymodel.RWIPosting
	if peers.hasHoldingPeers() {
		var err error
		if undeliveredValues, err = o.storedPostingsOf(tx, undelivered); err != nil {
			return Offer{}, err
		}
	}
	var wholeWordPartition []yacymodel.RWIPosting
	if peers.hasMissingHolders() {
		var err error
		if wholeWordPartition, err = o.postingsIn(tx, peers.wordPartition); err != nil {
			return Offer{}, err
		}
	}

	return peers.undeliveredOfferWith(
		undelivered,
		undeliveredValues,
		wholeWordPartition,
	), nil
}

func (o *Offers) storedPostingsOf(
	tx *vault.Txn,
	undelivered []undeliveredpostings.Posting,
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

func (o *Offers) postingsIn(
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

func (o *Offers) refreshOfferFor(
	tx *vault.Txn,
	undelivered []undeliveredpostings.Posting,
	peers wordPartitionPeers,
) (Offer, error) {
	wholeWordPartition, err := o.postingsIn(tx, peers.wordPartition)
	if err != nil {
		return Offer{}, err
	}

	return peers.refreshOfferWith(undelivered, wholeWordPartition), nil
}

func (o *Offers) DeliveredPostingsAmong(
	tx *vault.Txn,
	acceptedPostings []AcceptedPosting,
) ([]undeliveredpostings.Posting, error) {
	deliveredPostings := make([]undeliveredpostings.Posting, 0, len(acceptedPostings))
	for _, acceptedPosting := range acceptedPostings {
		stored, found, err := o.postings.PostingOf(
			tx,
			acceptedPosting.Sent.WordHash,
			acceptedPosting.Sent.URLHash,
		)
		if err != nil {
			return nil, err
		}
		if found && stored == acceptedPosting.Sent {
			deliveredPostings = append(deliveredPostings, acceptedPosting.Undelivered)
		}
	}

	return deliveredPostings, nil
}
