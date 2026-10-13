// Package postingtransfer sends the postings offered to one peer through the
// two-step wire exchange: a posting offer, then the metadata for the URLs the
// peer does not recognize. A posting counts as delivered only once both steps
// reach the peer. An offer fails when the peer gives no answer to it.
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
)

type URLDirectory interface {
	MetadataPerHash(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]yacymodel.URLMetadata, error)
}

type PostingCourier interface {
	Offer(
		ctx context.Context,
		recipient yacymodel.Seed,
		postings []yacymodel.RWIPosting,
	) postingcourier.Receipt
}

type URLMetadataCourier interface {
	Deliver(
		ctx context.Context,
		recipient yacymodel.Seed,
		metadata []yacymodel.URLMetadata,
	) urlmetadatacourier.Receipt
}

type Observer interface {
	ObservePostingOffer(outcome string, postings int)
	ObserveURLMetadataDelivery(outcome string, urls int)
	ObserveURLsUnknownToUs(urls int)
}

type OfferResult string

const (
	OfferAccepted OfferResult = "accepted"
	OfferDeclined OfferResult = "declined"
	OfferFailed   OfferResult = "failed"
)

type OfferAnswer struct {
	Result           OfferResult
	AcceptedPostings []yacymodel.RWIPosting
	RequestedPause   time.Duration
}

type Transfers struct {
	vault    *vault.Vault
	postings PostingCourier
	metadata URLMetadataCourier
	urls     URLDirectory
	observer Observer
}

func New(
	v *vault.Vault,
	postings PostingCourier,
	metadata URLMetadataCourier,
	urls URLDirectory,
	observer Observer,
) *Transfers {
	return &Transfers{
		vault:    v,
		postings: postings,
		metadata: metadata,
		urls:     urls,
		observer: observer,
	}
}

func (t *Transfers) Send(
	ctx context.Context,
	peer yacymodel.Seed,
	offeredPostings []yacymodel.RWIPosting,
) OfferAnswer {
	receipt := t.postings.Offer(ctx, peer, offeredPostings)
	t.observer.ObservePostingOffer(string(receipt.Outcome), len(offeredPostings))

	switch receipt.Outcome {
	case postingcourier.Accepted:
		undeliveredURLs := t.deliverURLMetadata(ctx, peer, receipt.URLsUnknownToPeer)

		return OfferAnswer{
			Result:           OfferAccepted,
			AcceptedPostings: postingsWithMetadataDelivered(offeredPostings, undeliveredURLs),
		}
	case postingcourier.Unreachable, postingcourier.Unaddressable:
		return OfferAnswer{Result: OfferFailed}
	default:
		return OfferAnswer{Result: OfferDeclined, RequestedPause: receipt.RequestedPause}
	}
}

func (t *Transfers) deliverURLMetadata(
	ctx context.Context,
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

	delivery := t.metadata.Deliver(ctx, peer, metadata)
	t.observer.ObserveURLMetadataDelivery(string(delivery.Outcome), len(metadata))

	if delivery.Outcome != urlmetadatacourier.Accepted {
		return urlsUnknownToPeer
	}

	return append(urlsUnknownToUs, delivery.URLsRejected...)
}

func (t *Transfers) metadataOfUnknownURLs(
	ctx context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, []yacymodel.URLHash, error) {
	var metadataPerHash map[yacymodel.URLHash]yacymodel.URLMetadata

	if err := t.vault.View(ctx, func(tx *vault.Txn) error {
		stored, err := t.urls.MetadataPerHash(tx, hashes)
		if err != nil {
			return err
		}
		metadataPerHash = stored

		return nil
	}); err != nil {
		return nil, nil, fmt.Errorf("read metadata of urls unknown to the peer: %w", err)
	}

	metadata := make([]yacymodel.URLMetadata, 0, len(hashes))

	missing := make([]yacymodel.URLHash, 0)

	for _, hash := range hashes {
		stored, ok := metadataPerHash[hash]
		if !ok {
			missing = append(missing, hash)

			continue
		}
		metadata = append(metadata, stored)
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
