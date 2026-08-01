package rwidistribution

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

type postingOfferResult struct {
	AcceptedPostings []yacymodel.RWIPosting
	RetryAfter       time.Duration
}

type postingOfferDelivery struct {
	postingCourier     postingCourier
	urlMetadataCourier urlMetadataCourier
	urls               urlmeta.URLDirectory
	observer           PostingOfferCycleObserver
}

func (d *postingOfferDelivery) Offer(ctx context.Context, offer postingOffer) postingOfferResult {
	endpoint, ok := offer.Peer.NetworkAddress()
	if !ok {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without network address",
			slog.String("peer", offer.Peer.Hash.String()),
		)
		d.observer.ObservePostingOffer(string(postingOfferUnaddressable), len(offer.Postings))

		return postingOfferResult{}
	}

	receipt := d.postingCourier.Offer(ctx, endpoint, offer)
	d.observer.ObservePostingOffer(string(receipt.Outcome), len(offer.Postings))

	if receipt.Outcome != postingOfferAccepted {
		return postingOfferResult{RetryAfter: receipt.RetryAfter}
	}

	urlsWithoutMetadata := d.deliverURLMetadata(ctx, endpoint, offer, receipt.URLsUnknownToPeer)
	accepted := offer.Postings
	if len(urlsWithoutMetadata) > 0 {
		accepted = postingsWithMetadataDelivered(offer.Postings, urlsWithoutMetadata)
	}

	return postingOfferResult{AcceptedPostings: accepted, RetryAfter: receipt.RetryAfter}
}

func (d *postingOfferDelivery) deliverURLMetadata(
	ctx context.Context,
	endpoint string,
	offer postingOffer,
	unknownToPeer []yacymodel.URLHash,
) []yacymodel.URLHash {
	if len(unknownToPeer) == 0 {
		return nil
	}

	metadata, err := d.urls.MetadataByHash(ctx, unknownToPeer)
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
		d.observer.ObserveURLMetadataDelivery(string(urlMetadataUnavailable), len(unknownToPeer))

		return unknownToPeer
	}

	delivery := d.urlMetadataCourier.Deliver(ctx, endpoint, offer.Peer.Hash, metadata)
	d.observer.ObserveURLMetadataDelivery(string(delivery.Outcome), len(metadata))

	if delivery.Outcome != urlMetadataAccepted {
		return unknownToPeer
	}

	return delivery.URLsRejected
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
