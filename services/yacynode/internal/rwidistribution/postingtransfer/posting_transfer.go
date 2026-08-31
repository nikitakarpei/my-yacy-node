// Package postingtransfer sends the postings offered to one peer through the
// two-step wire exchange: a posting offer, then the metadata for the URLs the
// peer does not recognize. A posting counts as delivered only once both steps
// reach the peer.
package postingtransfer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type Observer interface {
	ObservePostingOffer(outcome string, postings int)
	ObserveURLMetadataDelivery(outcome string, urls int)
	ObserveURLsUnknownToUs(urls int)
}

type OfferAnswer struct {
	Accepted         bool
	AcceptedPostings []yacymodel.RWIPosting
	RequestedPause   time.Duration
}

type PostingTransfers struct {
	vault    *vault.Vault
	postings postingcourier.Courier
	metadata urlmetadatacourier.Courier
	urls     urlmeta.URLDirectory
	observer Observer
}

func New(
	v *vault.Vault,
	postings postingcourier.Courier,
	metadata urlmetadatacourier.Courier,
	urls urlmeta.URLDirectory,
	observer Observer,
) *PostingTransfers {
	return &PostingTransfers{
		vault:    v,
		postings: postings,
		metadata: metadata,
		urls:     urls,
		observer: observer,
	}
}

func (t *PostingTransfers) Send(
	ctx context.Context,
	peer yacymodel.Seed,
	offeredPostings []yacymodel.RWIPosting,
) OfferAnswer {
	endpoint, ok := peer.NetworkAddress()
	if !ok {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without network address",
			slog.String("peer", peer.Hash.String()),
		)
		t.observer.ObservePostingOffer(string(postingcourier.Unaddressable), len(offeredPostings))

		return OfferAnswer{}
	}

	receipt := t.postings.Offer(ctx, endpoint, peer, offeredPostings)
	t.observer.ObservePostingOffer(string(receipt.Outcome), len(offeredPostings))

	if receipt.Outcome != postingcourier.Accepted {
		return OfferAnswer{RequestedPause: receipt.RequestedPause}
	}

	undeliveredURLs := t.deliverURLMetadata(ctx, endpoint, peer, receipt.URLsUnknownToPeer)

	return OfferAnswer{
		Accepted:         true,
		AcceptedPostings: postingsWithMetadataDelivered(offeredPostings, undeliveredURLs),
		RequestedPause:   receipt.RequestedPause,
	}
}

func (t *PostingTransfers) deliverURLMetadata(
	ctx context.Context,
	endpoint string,
	peer yacymodel.Seed,
	urlsUnknownToPeer []yacymodel.URLHash,
) []yacymodel.URLHash {
	if len(urlsUnknownToPeer) == 0 {
		return nil
	}

	metadata, urlsUnknownToUs, err := t.metadataOfUnknownURLs(ctx, urlsUnknownToPeer)
	if err != nil {
		slog.WarnContext(
			ctx,
			"unknown url metadata not read",
			slog.String("peer", peer.Hash.String()),
			slog.Any("error", err),
		)

		return urlsUnknownToPeer
	}
	if len(urlsUnknownToUs) > 0 {
		slog.WarnContext(
			ctx,
			"url metadata absent for peer's unknown urls",
			slog.String("peer", peer.Hash.String()),
			slog.Int("unknownToPeer", len(urlsUnknownToPeer)),
			slog.Int("unknownToUs", len(urlsUnknownToUs)),
		)
		t.observer.ObserveURLsUnknownToUs(len(urlsUnknownToUs))
	}
	if len(metadata) == 0 {
		return urlsUnknownToUs
	}

	delivery := t.metadata.Deliver(ctx, endpoint, peer.Hash, metadata)
	t.observer.ObserveURLMetadataDelivery(string(delivery.Outcome), len(metadata))

	if delivery.Outcome != urlmetadatacourier.Accepted {
		return urlsUnknownToPeer
	}

	return append(urlsUnknownToUs, delivery.URLsRejected...)
}

func (t *PostingTransfers) metadataOfUnknownURLs(
	ctx context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, []yacymodel.URLHash, error) {
	var metadata []yacymodel.URLMetadata

	var missing []yacymodel.URLHash

	if err := t.vault.View(ctx, func(tx *vault.Txn) error {
		stored, err := t.urls.MetadataByHash(tx, hashes)
		if err != nil {
			return err
		}
		absent, err := t.urls.MissingURLs(tx, hashes)
		if err != nil {
			return err
		}
		metadata, missing = stored, absent

		return nil
	}); err != nil {
		return nil, nil, fmt.Errorf("read metadata of urls unknown to the peer: %w", err)
	}

	return metadata, missing, nil
}

func postingsWithMetadataDelivered(
	postings []yacymodel.RWIPosting,
	undeliveredURLs []yacymodel.URLHash,
) []yacymodel.RWIPosting {
	withoutMetadata := make(map[yacymodel.URLHash]struct{}, len(undeliveredURLs))
	for _, url := range undeliveredURLs {
		withoutMetadata[url] = struct{}{}
	}

	keptPostings := make([]yacymodel.RWIPosting, 0, len(postings))
	for _, posting := range postings {
		if _, excluded := withoutMetadata[posting.URLHash]; excluded {
			continue
		}
		keptPostings = append(keptPostings, posting)
	}

	return keptPostings
}
