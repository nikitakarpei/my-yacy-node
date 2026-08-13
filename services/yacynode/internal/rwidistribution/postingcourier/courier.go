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

type Receipt struct {
	Outcome           Outcome
	RequestedPause    time.Duration
	URLsUnknownToPeer []yacymodel.URLHash
}

type Courier interface {
	Offer(
		ctx context.Context,
		endpoint string,
		recipient yacymodel.Seed,
		postings []yacymodel.RWIPosting,
	) Receipt
}

type httpCourier struct {
	exchange    peerwire.MessageExchange
	networkName string
	self        yacymodel.Hash
}

func New(
	exchange peerwire.MessageExchange,
	networkName string,
	self yacymodel.Hash,
) Courier {
	return httpCourier{exchange: exchange, networkName: networkName, self: self}
}

func (c httpCourier) Offer(
	ctx context.Context,
	endpoint string,
	recipient yacymodel.Seed,
	postings []yacymodel.RWIPosting,
) Receipt {
	resp, err := c.postTransferRWI(ctx, endpoint, recipient, postings)
	if err != nil {
		slog.WarnContext(
			ctx,
			"posting offer failed",
			slog.String("peer", recipient.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return Receipt{Outcome: Unreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK:
		return Receipt{
			Outcome:           Accepted,
			URLsUnknownToPeer: resp.UnknownURL,
		}
	case yacyproto.ResultBusy, yacyproto.ResultNotGranted:
		return Receipt{Outcome: Deferred, RequestedPause: resp.Pause}
	case yacyproto.ResultTooHighLoad:
		slog.WarnContext(
			ctx,
			"posting offer rejected by loaded peer",
			slog.String("peer", recipient.Hash.String()),
			slog.String("endpoint", endpoint),
		)

		return Receipt{Outcome: Overloaded}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", recipient.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}

func (c httpCourier) postTransferRWI(
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
