// Package postingoffer works out, for each posting due an offer, which peers
// to offer it to under the DHT, how many of them must accept for the posting
// to be at redundancy, and which ledger entries the DHT no longer justifies.
package postingoffer

import (
	"context"
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const replicaLedgerUnread = "read replica ledger: %w"

type PostingOffer struct {
	Posting           yacymodel.RWIPosting
	PostingPosition   yacymodel.DHTRingPosition
	Peers             []yacymodel.Seed
	AcceptancesNeeded int
	StaleHolders      []yacymodel.Hash
}

type Observer interface {
	ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int)
}

type Reachability interface {
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
	IsRecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type ReplicaEligibility interface {
	EligiblePeers(peers []yacymodel.Seed) []yacymodel.Seed
}

type PostingOffers struct {
	vault        *vault.Vault
	schedule     *postingofferschedule.Schedule
	replicas     *postingreplicas.Replicas
	postings     rwipostings.PostingIndex
	reachability Reachability
	eligibility  ReplicaEligibility
	observer     Observer
	partitions   yacymodel.DHTRingPartitions
	self         yacymodel.Hash
	redundancy   int
}

//nolint:revive // argument-limit: ten explicit, independently-meaningful collaborators
func New(
	v *vault.Vault,
	schedule *postingofferschedule.Schedule,
	replicas *postingreplicas.Replicas,
	postings rwipostings.PostingIndex,
	reachability Reachability,
	eligibility ReplicaEligibility,
	observer Observer,
	partitions yacymodel.DHTRingPartitions,
	self yacymodel.Hash,
	redundancy int,
) *PostingOffers {
	return &PostingOffers{
		vault:        v,
		schedule:     schedule,
		replicas:     replicas,
		postings:     postings,
		reachability: reachability,
		eligibility:  eligibility,
		observer:     observer,
		partitions:   partitions,
		self:         self,
		redundancy:   redundancy,
	}
}

func (o *PostingOffers) DueNow(
	ctx context.Context,
	limit int,
	reachablePeers []yacymodel.Seed,
) ([]PostingOffer, []postingidentity.Identity, error) {
	duePostings, err := o.schedule.DuePostings(ctx, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("read due postings: %w", err)
	}

	acceptingPeers := peersAcceptingRemoteIndex(reachablePeers)
	o.observer.ObservePeersAcceptingRemoteIndexPerDHTRingSector(
		yacymodel.SeedsPerDHTRingSector(acceptingPeers),
	)

	var (
		offers       []PostingOffer
		gonePostings []postingidentity.Identity
	)
	err = o.vault.View(ctx, func(tx *vault.Txn) error {
		postings, gone, err := o.storedPostings(tx, duePostings)
		if err != nil {
			return err
		}

		dueOffers := make([]PostingOffer, 0, len(postings))
		for _, posting := range postings {
			offer, err := o.offerFor(ctx, tx, posting, acceptingPeers)
			if err != nil {
				return err
			}
			dueOffers = append(dueOffers, offer)
		}
		offers, gonePostings = dueOffers, gone

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return offers, gonePostings, nil
}

func (o *PostingOffers) storedPostings(
	tx *vault.Txn,
	duePostings []postingidentity.Identity,
) ([]yacymodel.RWIPosting, []postingidentity.Identity, error) {
	var (
		postings     []yacymodel.RWIPosting
		gonePostings []postingidentity.Identity
	)
	for _, identity := range duePostings {
		posting, found, err := o.postings.PostingOf(tx, identity.Word, identity.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("read posting: %w", err)
		}
		if !found {
			gonePostings = append(gonePostings, identity)

			continue
		}
		postings = append(postings, posting)
	}

	return postings, gonePostings, nil
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

func (o *PostingOffers) offerFor(
	ctx context.Context,
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	acceptingPeers []yacymodel.Seed,
) (PostingOffer, error) {
	identity := postingidentity.IdentityOf(posting.WordHash, posting.URLHash)
	recordedHolders, err := o.replicas.HoldersOf(tx, identity)
	if err != nil {
		return PostingOffer{}, fmt.Errorf(replicaLedgerUnread, err)
	}

	position := yacymodel.DHTRingPositionOfPosting(posting, o.partitions)
	window := replicaWindowFor(position, acceptingPeers, o.self, o.redundancy)
	holders := o.replicaHoldersFrom(ctx, recordedHolders, window)

	if len(holders.insideWindow) >= window.requiredReplicas {
		return refreshOfferFor(posting, acceptingPeers, holders, window), nil
	}

	return o.replicaOfferFor(posting, acceptingPeers, holders, window), nil
}

func (o *PostingOffers) replicaHoldersFrom(
	ctx context.Context,
	recordedHolders []yacymodel.Hash,
	window replicaWindow,
) replicaHolders {
	var holders replicaHolders
	for _, peer := range recordedHolders {
		switch {
		case !o.reachability.IsReachable(ctx, peer):
			if o.reachability.IsRecentlyReachable(ctx, peer) {
				holders.insideWindow = append(holders.insideWindow, peer)
			} else {
				holders.expired = append(holders.expired, peer)
			}
		case window.contains(peer):
			holders.insideWindow = append(holders.insideWindow, peer)
		default:
			holders.outsideWindow = append(holders.outsideWindow, peer)
		}
	}

	return holders
}

func refreshOfferFor(
	posting yacymodel.RWIPosting,
	acceptingPeers []yacymodel.Seed,
	holders replicaHolders,
	window replicaWindow,
) PostingOffer {
	return PostingOffer{
		Posting:         posting,
		PostingPosition: window.position,
		Peers:           holders.peersHoldingReplica(acceptingPeers),
		StaleHolders:    slices.Concat(holders.expired, holders.outsideWindow),
	}
}

func (o *PostingOffers) replicaOfferFor(
	posting yacymodel.RWIPosting,
	acceptingPeers []yacymodel.Seed,
	holders replicaHolders,
	window replicaWindow,
) PostingOffer {
	missingReplicas := window.requiredReplicas - len(holders.insideWindow)
	eligiblePeers := o.eligibility.EligiblePeers(acceptingPeers)

	return PostingOffer{
		Posting:         posting,
		PostingPosition: window.position,
		Peers: yacymodel.SeedsClosestToDHTRingPosition(
			holders.peersMissingReplica(eligiblePeers), window.position, missingReplicas,
		),
		AcceptancesNeeded: missingReplicas,
		StaleHolders:      holders.expired,
	}
}
