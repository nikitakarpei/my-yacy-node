package rwidistribution

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type postingCourier interface {
	Offer(ctx context.Context, endpoint string, offer postingOffer) postingOfferReceipt
}

type urlMetadataCourier interface {
	Deliver(
		ctx context.Context,
		endpoint string,
		peer yacymodel.Hash,
		metadata []yacymodel.URLMetadata,
	) urlMetadataReceipt
}

type postingOfferCycle struct {
	reader             *postingReplicationReader
	postingCourier     postingCourier
	urlMetadataCourier urlMetadataCourier
	urls               urlmeta.URLDirectory
	schedule           *postingOfferSchedule
	ledger             *replicaLedger
	roster             peerroster.Roster
	observer           PostingOfferCycleObserver
	cadence            postingOfferCadence
	now                func() time.Time
	postingsPerCycle   int
	cycleInterval      time.Duration
	minReachablePeers  int
}

func (c *postingOfferCycle) Run(ctx context.Context) {
	c.runCycle(ctx)

	ticker := time.NewTicker(c.cycleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runCycle(ctx)
		}
	}
}

func (c *postingOfferCycle) runCycle(ctx context.Context) {
	c.observeOldestDuePostingAge(ctx)

	reachable := len(c.roster.ReachablePeers(ctx))
	if reachable < c.minReachablePeers {
		slog.DebugContext(
			ctx,
			"distribution cycle skipped: too few reachable peers",
			slog.Int("reachablePeers", reachable),
			slog.Int("minReachablePeers", c.minReachablePeers),
		)
		c.observer.ObserveCycleSkipped()

		return
	}

	due, err := c.reader.DueReplication(ctx, c.postingsPerCycle)
	if err != nil {
		slog.ErrorContext(ctx, "posting replication not read", slog.Any("error", err))

		return
	}
	c.observer.ObservePostingsDue(len(due.Postings))
	c.observer.ObservePostingsGone(len(due.Gone))
	for _, identity := range due.Gone {
		slog.DebugContext(ctx, "due posting gone from index",
			slog.String("word", identity.Word.String()),
			slog.String("url", identity.URL.String()))
	}

	byPeer := newPostingOffersByPeer()
	for _, replication := range due.Postings {
		for _, seed := range replication.SeedsMissingCopy {
			if !byPeer.Full(seed.Hash) {
				byPeer.Add(seed, replication.Posting)
			}
		}
	}

	tally := newPostingOfferTally()
	for _, offer := range byPeer.Offers() {
		c.offerToPeer(ctx, offer, tally)
	}

	c.dropReplicasOfPeersNoLongerResponsible(ctx, due.Postings)
	c.rescheduleOfferedPostings(ctx, due.Postings, tally)
}

func (c *postingOfferCycle) offerToPeer(
	ctx context.Context,
	offer postingOffer,
	tally *postingOfferTally,
) {
	endpoint, ok := offer.Peer.NetworkAddress()
	if !ok {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without network address",
			slog.String("peer", offer.Peer.Hash.String()),
		)
		c.observer.ObservePostingOffer(string(postingOfferUnaddressable), len(offer.Postings))

		return
	}

	receipt := c.postingCourier.Offer(ctx, endpoint, offer)
	c.observer.ObservePostingOffer(string(receipt.Outcome), len(offer.Postings))
	tally.RecordOffer(offer, receipt)

	if receipt.Outcome != postingOfferAccepted {
		return
	}

	urlsWithoutMetadata := c.deliverURLMetadata(ctx, endpoint, offer, receipt.URLsUnknownToPeer)
	tally.RecordURLsWithoutMetadata(offer, urlsWithoutMetadata)
	c.recordAcceptedPostings(ctx, offer, urlsWithoutMetadata)
}

func (c *postingOfferCycle) deliverURLMetadata(
	ctx context.Context,
	endpoint string,
	offer postingOffer,
	unknownToPeer []yacymodel.URLHash,
) []yacymodel.URLHash {
	if len(unknownToPeer) == 0 {
		return nil
	}

	metadata, err := c.urls.MetadataByHash(ctx, unknownToPeer)
	if err != nil {
		slog.WarnContext(
			ctx,
			"unknown url metadata not read",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.Any("error", err),
		)

		return unknownToPeer
	}

	if len(metadata) != len(unknownToPeer) {
		slog.WarnContext(
			ctx,
			"url metadata incomplete for peer's unknown urls",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.Int("unknown", len(unknownToPeer)),
			slog.Int("found", len(metadata)),
		)
		c.observer.ObserveURLMetadataDelivery(string(urlMetadataUnavailable), len(unknownToPeer))

		return unknownToPeer
	}

	delivery := c.urlMetadataCourier.Deliver(ctx, endpoint, offer.Peer.Hash, metadata)
	c.observer.ObserveURLMetadataDelivery(string(delivery.Outcome), len(metadata))

	if delivery.Outcome != urlMetadataAccepted {
		return unknownToPeer
	}

	return delivery.URLsRejected
}

func (c *postingOfferCycle) recordAcceptedPostings(
	ctx context.Context,
	offer postingOffer,
	urlsWithoutMetadata []yacymodel.URLHash,
) {
	replicated := offer
	if len(urlsWithoutMetadata) > 0 {
		replicated.Postings = postingsWithMetadataDelivered(offer.Postings, urlsWithoutMetadata)
		if len(replicated.Postings) == 0 {
			return
		}
	}

	if err := c.ledger.RecordAccepted(ctx, replicated); err != nil {
		slog.WarnContext(ctx, "replicas not recorded",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.Any("error", err))
	}
}

func postingsWithMetadataDelivered(
	postings []yacymodel.RWIPosting,
	urlsWithoutMetadata []yacymodel.URLHash,
) []yacymodel.RWIPosting {
	without := make(map[yacymodel.URLHash]struct{}, len(urlsWithoutMetadata))
	for _, url := range urlsWithoutMetadata {
		without[url] = struct{}{}
	}

	kept := make([]yacymodel.RWIPosting, 0, len(postings))
	for _, posting := range postings {
		if _, excluded := without[posting.URLHash]; excluded {
			continue
		}
		kept = append(kept, posting)
	}

	return kept
}

func (c *postingOfferCycle) observeOldestDuePostingAge(ctx context.Context) {
	oldest, found, err := c.schedule.OldestDueAt(ctx)
	if err != nil {
		slog.WarnContext(ctx, "oldest due posting not read", slog.Any("error", err))

		return
	}
	if !found {
		return
	}

	age := c.now().Sub(oldest)
	if age < 0 {
		age = 0
	}
	c.observer.ObserveOldestDuePostingAge(age)
}

func (c *postingOfferCycle) dropReplicasOfPeersNoLongerResponsible(
	ctx context.Context,
	postings []postingReplication,
) {
	for _, replication := range postings {
		if len(replication.PeerHashesNoLongerResponsible) == 0 {
			continue
		}

		word, url := replication.Posting.WordHash, replication.Posting.URLHash
		dropped, err := c.ledger.RecordDropped(
			ctx, word, url, replication.PeerHashesNoLongerResponsible,
		)
		if err != nil {
			slog.WarnContext(ctx, "stale replicas not dropped",
				slog.String("word", word.String()),
				slog.String("url", url.String()),
				slog.Any("error", err))

			continue
		}
		if dropped > 0 {
			c.observer.ObserveLedgerPrune(dropped)
		}
	}
}

func (c *postingOfferCycle) rescheduleOfferedPostings(
	ctx context.Context,
	postings []postingReplication,
	tally *postingOfferTally,
) {
	now := c.now()
	for _, replication := range postings {
		id := postingIdentity{Word: replication.Posting.WordHash, URL: replication.Posting.URLHash}
		replicated := replication.CopiesNeeded == 0 ||
			tally.AcceptedCopies(id) >= replication.CopiesNeeded
		at := c.cadence.NextDue(now, replicated, tally.RetryAfter(id))

		if err := c.schedule.Reschedule(ctx, id.Word, id.URL, at); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", id.Word.String()),
				slog.String("url", id.URL.String()),
				slog.Any("error", err))
		}
	}
}
