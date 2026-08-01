package rwidistribution

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type postingOfferReceipt struct {
	Outcome           postingOfferOutcome
	RetryAfter        time.Duration
	URLsUnknownToPeer []yacymodel.URLHash
}

type httpPostingCourier struct {
	exchange    peerMessageExchange
	networkName string
	self        yacymodel.Hash
}

func (c httpPostingCourier) Offer(
	ctx context.Context,
	endpoint string,
	offer postingOffer,
) postingOfferReceipt {
	resp, err := c.postTransferRWI(ctx, endpoint, offer)
	if err != nil {
		slog.WarnContext(
			ctx,
			"posting offer failed",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return postingOfferReceipt{Outcome: postingOfferUnreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK:
		return postingOfferReceipt{
			Outcome:           postingOfferAccepted,
			URLsUnknownToPeer: resp.UnknownURL,
		}
	case yacyproto.ResultBusy, yacyproto.ResultNotGranted:
		return postingOfferReceipt{Outcome: postingOfferDeferred, RetryAfter: resp.Pause}
	case yacyproto.ResultTooHighLoad:
		slog.WarnContext(
			ctx,
			"posting offer rejected by loaded peer",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
		)

		return postingOfferReceipt{Outcome: postingOfferOverloaded}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)

		return postingOfferReceipt{Outcome: postingOfferRefused}
	}
}

func (c httpPostingCourier) postTransferRWI(
	ctx context.Context,
	endpoint string,
	offer postingOffer,
) (yacyproto.TransferRWIResponse, error) {
	req := yacyproto.TransferRWIRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      offer.Peer.Hash,
		WordCount:   distinctWordCount(offer.Postings),
		EntryCount:  len(offer.Postings),
		Indexes:     offer.Postings,
		Key:         yacyproto.TransferRWIKey(len(offer.Postings)),
	}

	msg, err := c.exchange.Exchange(ctx, endpoint, yacyproto.PathTransferRWI, req.Form())
	if err != nil {
		return yacyproto.TransferRWIResponse{}, err
	}

	resp, err := yacyproto.ParseTransferRWIResponse(msg)
	if err != nil {
		return yacyproto.TransferRWIResponse{}, fmt.Errorf("%w: %w", errPeerRequest, err)
	}

	return resp, nil
}

func distinctWordCount(postings []yacymodel.RWIPosting) int {
	words := make(map[yacymodel.Hash]struct{}, len(postings))
	for _, posting := range postings {
		words[posting.WordHash] = struct{}{}
	}

	return len(words)
}
