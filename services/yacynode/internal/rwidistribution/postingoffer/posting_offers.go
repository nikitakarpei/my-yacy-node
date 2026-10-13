// Package postingoffer works out, for each posting due an offer, which peers
// to offer it to under the DHT, how many peers must accept for the posting to
// be at redundancy, how long peers in cooldown make it wait, and which ledger
// entries the DHT no longer justifies.
package postingoffer

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
)

const replicaLedgerUnread = "read replica ledger: %w"

type PostingOffer struct {
	Posting                 yacymodel.RWIPosting
	PostingPosition         yacymodel.DHTRingPosition
	Peers                   []yacymodel.Seed
	AcceptancesNeeded       int
	PauseForPeersInCooldown time.Duration
	StaleHolders            []yacymodel.Hash
}

type Observer interface {
	ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int)
}

type PostingIndex interface {
	PostingOf(
		tx *vault.Txn,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
}

type Reachability interface {
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
	IsRecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type Cooldowns interface {
	CooldownLeftOf(peer yacymodel.Hash) time.Duration
}

type Config struct {
	Partitions yacymodel.DHTRingPartitions
	Self       yacymodel.Hash
	Redundancy int
}

type PostingOffers struct {
	vault        *vault.Vault
	schedule     *postingofferschedule.Schedule
	replicas     *postingreplicas.Replicas
	postings     PostingIndex
	reachability Reachability
	cooldowns    Cooldowns
	observer     Observer
	config       Config
}

//nolint:revive // argument-limit: seven explicit, independently-meaningful collaborators and the config
func New(
	v *vault.Vault,
	schedule *postingofferschedule.Schedule,
	replicas *postingreplicas.Replicas,
	postings PostingIndex,
	reachability Reachability,
	cooldowns Cooldowns,
	observer Observer,
	config Config,
) *PostingOffers {
	return &PostingOffers{
		vault:        v,
		schedule:     schedule,
		replicas:     replicas,
		postings:     postings,
		reachability: reachability,
		cooldowns:    cooldowns,
		observer:     observer,
		config:       config,
	}
}

func (o *PostingOffers) ReportPeersAcceptingRemoteIndex(reachablePeers []yacymodel.Seed) {
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

func (o *PostingOffers) DueNowIn(
	ctx context.Context,
	sector yacymodel.DHTRingSector,
	limit int,
	reachablePeers []yacymodel.Seed,
) ([]PostingOffer, error) {
	acceptingPeerRing := yacymodel.SeedRingFrom(peersAcceptingRemoteIndex(reachablePeers))

	var offers []PostingOffer
	err := o.vault.View(ctx, func(tx *vault.Txn) error {
		duePostings, err := o.schedule.DuePostingsIn(tx, sector, limit)
		if err != nil {
			return err
		}
		postings, err := o.storedPostings(ctx, tx, duePostings)
		if err != nil {
			return err
		}

		offers = make([]PostingOffer, 0, len(postings))
		for _, posting := range postings {
			offer, err := o.offerFor(ctx, tx, posting, acceptingPeerRing)
			if err != nil {
				return err
			}
			offers = append(offers, offer)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return offers, nil
}

func (o *PostingOffers) storedPostings(
	ctx context.Context,
	tx *vault.Txn,
	duePostings []postingidentity.Identity,
) ([]yacymodel.RWIPosting, error) {
	postings := make([]yacymodel.RWIPosting, 0, len(duePostings))
	for _, identity := range duePostings {
		posting, found, err := o.postings.PostingOf(tx, identity.Word, identity.URL)
		if err != nil {
			return nil, fmt.Errorf("read posting: %w", err)
		}
		if !found {
			slog.WarnContext(ctx, "due posting missing from the index",
				slog.String("word", identity.Word.String()),
				slog.String("url", identity.URL.String()))

			continue
		}
		postings = append(postings, posting)
	}

	return postings, nil
}

func (o *PostingOffers) offerFor(
	ctx context.Context,
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	acceptingPeers yacymodel.SeedRing,
) (PostingOffer, error) {
	identity := postingidentity.IdentityOf(posting)
	recordedHolders, err := o.replicas.HoldersOf(tx, identity)
	if err != nil {
		return PostingOffer{}, fmt.Errorf(replicaLedgerUnread, err)
	}

	position := yacymodel.DHTRingPositionOfPosting(posting, o.config.Partitions)
	window := replicaWindowFor(position, acceptingPeers, o.config.Self, o.config.Redundancy)
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
	acceptingPeers yacymodel.SeedRing,
	holders replicaHolders,
	window replicaWindow,
) PostingOffer {
	return PostingOffer{
		Posting:         posting,
		PostingPosition: window.position,
		Peers: holders.peersHoldingReplica(
			acceptingPeers.SeedsClockwiseFrom(window.position),
		),
		StaleHolders: slices.Concat(holders.expired, holders.outsideWindow),
	}
}

func (o *PostingOffers) replicaOfferFor(
	posting yacymodel.RWIPosting,
	acceptingPeers yacymodel.SeedRing,
	holders replicaHolders,
	window replicaWindow,
) PostingOffer {
	missingReplicas := window.requiredReplicas - len(holders.insideWindow)
	peersMissingReplica := missingPeersFrom(
		holders.closestPeersMissingReplica(acceptingPeers, window.position, missingReplicas),
		o.cooldowns,
	)

	return PostingOffer{
		Posting:                 posting,
		PostingPosition:         window.position,
		Peers:                   peersMissingReplica.outOfCooldown,
		AcceptancesNeeded:       missingReplicas,
		PauseForPeersInCooldown: peersMissingReplica.longestCooldownLeft,
		StaleHolders:            holders.expired,
	}
}
