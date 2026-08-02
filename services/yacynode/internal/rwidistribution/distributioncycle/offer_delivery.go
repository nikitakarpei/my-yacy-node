package distributioncycle

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type OfferReceipt struct {
	AcceptedPostings []yacymodel.RWIPosting
	Backoff          time.Duration
}

type OfferDelivery struct {
	postingCourier     postingcourier.PostingCourier
	urlMetadataCourier urlmetadatacourier.URLMetadataCourier
	urls               urlmeta.URLDirectory
	observer           Observer
}

func NewOfferDelivery(
	postingCourier postingcourier.PostingCourier,
	urlMetadataCourier urlmetadatacourier.URLMetadataCourier,
	urls urlmeta.URLDirectory,
	observer Observer,
) *OfferDelivery {
	return &OfferDelivery{
		postingCourier:     postingCourier,
		urlMetadataCourier: urlMetadataCourier,
		urls:               urls,
		observer:           observer,
	}
}

func (d *OfferDelivery) Offer(ctx context.Context, peerOffer offer) OfferReceipt {
	endpoint, ok := peerOffer.Peer.NetworkAddress()
	if !ok {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without network address",
			slog.String("peer", peerOffer.Peer.Hash.String()),
		)
		d.observer.ObservePostingOffer(
			string(postingcourier.Unaddressable), len(peerOffer.Postings),
		)

		return OfferReceipt{}
	}

	receipt := d.postingCourier.Offer(ctx, endpoint, peerOffer.Peer, peerOffer.Postings)
	d.observer.ObservePostingOffer(string(receipt.Outcome), len(peerOffer.Postings))

	if receipt.Outcome != postingcourier.Accepted {
		return OfferReceipt{Backoff: receipt.RetryAfter}
	}

	undelivered := d.deliverURLMetadata(ctx, endpoint, peerOffer, receipt.URLsUnknownToPeer)

	return OfferReceipt{
		AcceptedPostings: postingsWithMetadataDelivered(peerOffer.Postings, undelivered),
		Backoff:          receipt.RetryAfter,
	}
}

func (d *OfferDelivery) deliverURLMetadata(
	ctx context.Context,
	endpoint string,
	peerOffer offer,
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
			slog.String("peer", peerOffer.Peer.Hash.String()),
			slog.Any("error", err),
		)

		return unknownToPeer
	}

	unknownToUs, err := d.urls.MissingURLs(ctx, unknownToPeer)
	if err != nil {
		slog.WarnContext(
			ctx,
			"unknown url metadata presence not read",
			slog.String("peer", peerOffer.Peer.Hash.String()),
			slog.Any("error", err),
		)

		return unknownToPeer
	}
	if len(unknownToUs) > 0 {
		slog.WarnContext(
			ctx,
			"url metadata absent for peer's unknown urls",
			slog.String("peer", peerOffer.Peer.Hash.String()),
			slog.Int("unknownToPeer", len(unknownToPeer)),
			slog.Int("unknownToUs", len(unknownToUs)),
		)
		d.observer.ObserveURLMetadataDelivery(
			string(urlmetadatacourier.Unavailable), len(unknownToUs),
		)
	}
	if len(metadata) == 0 {
		return unknownToUs
	}

	delivery := d.urlMetadataCourier.Deliver(ctx, endpoint, peerOffer.Peer.Hash, metadata)
	d.observer.ObserveURLMetadataDelivery(string(delivery.Outcome), len(metadata))

	if delivery.Outcome != urlmetadatacourier.Accepted {
		return unknownToPeer
	}

	return append(unknownToUs, delivery.URLsRejected...)
}

func postingsWithMetadataDelivered(
	postings []yacymodel.RWIPosting,
	undelivered []yacymodel.URLHash,
) []yacymodel.RWIPosting {
	without := make(map[yacymodel.URLHash]struct{}, len(undelivered))
	for _, url := range undelivered {
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
