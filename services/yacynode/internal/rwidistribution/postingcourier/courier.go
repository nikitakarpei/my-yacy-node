// Package postingcourier sends stored postings to a peer over the transferRWI
// protocol endpoint and reports how the peer responded.
package postingcourier

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type PostingReceipt struct {
	Outcome           Outcome
	RetryAfter        time.Duration
	URLsUnknownToPeer []yacymodel.URLHash
}

type PostingCourier interface {
	Offer(
		ctx context.Context,
		endpoint string,
		recipient yacymodel.Seed,
		postings []yacymodel.RWIPosting,
	) PostingReceipt
}

type httpPostingCourier struct {
	exchange    peerwire.MessageExchange
	networkName string
	self        yacymodel.Hash
}

func New(
	exchange peerwire.MessageExchange,
	networkName string,
	self yacymodel.Hash,
) PostingCourier {
	return httpPostingCourier{exchange: exchange, networkName: networkName, self: self}
}

func (c httpPostingCourier) Offer(
	ctx context.Context,
	endpoint string,
	recipient yacymodel.Seed,
	postings []yacymodel.RWIPosting,
) PostingReceipt {
	resp, err := c.postTransferRWI(ctx, endpoint, recipient, postings)
	if err != nil {
		slog.WarnContext(
			ctx,
			"posting offer failed",
			slog.String("peer", recipient.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return PostingReceipt{Outcome: Unreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK:
		return PostingReceipt{
			Outcome:           Accepted,
			URLsUnknownToPeer: resp.UnknownURL,
		}
	case yacyproto.ResultBusy, yacyproto.ResultNotGranted:
		return PostingReceipt{Outcome: Deferred, RetryAfter: resp.Pause}
	case yacyproto.ResultTooHighLoad:
		slog.WarnContext(
			ctx,
			"posting offer rejected by loaded peer",
			slog.String("peer", recipient.Hash.String()),
			slog.String("endpoint", endpoint),
		)

		return PostingReceipt{Outcome: Overloaded}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", recipient.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)

		return PostingReceipt{Outcome: Refused}
	}
}

func (c httpPostingCourier) postTransferRWI(
	ctx context.Context,
	endpoint string,
	recipient yacymodel.Seed,
	postings []yacymodel.RWIPosting,
) (yacyproto.TransferRWIResponse, error) {
	req := yacyproto.TransferRWIRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      recipient.Hash,
		WordCount:   distinctWordCount(postings),
		EntryCount:  len(postings),
		Indexes:     postings,
		Key:         yacyproto.TransferRWIKey(len(postings)),
	}

	msg, err := c.exchange.Exchange(ctx, endpoint, yacyproto.PathTransferRWI, req.Form())
	if err != nil {
		return yacyproto.TransferRWIResponse{}, err
	}

	resp, err := yacyproto.ParseTransferRWIResponse(msg)
	if err != nil {
		return yacyproto.TransferRWIResponse{}, fmt.Errorf("parse transferRWI response: %w", err)
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
