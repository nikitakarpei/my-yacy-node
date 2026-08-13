// Package urlmetadatacourier delivers URL metadata to a peer over the
// transferURL protocol endpoint and reports how the peer responded.
package urlmetadatacourier

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type Receipt struct {
	Outcome      Outcome
	URLsRejected []yacymodel.URLHash
}

type Courier interface {
	Deliver(
		ctx context.Context,
		endpoint string,
		peer yacymodel.Hash,
		metadata []yacymodel.URLMetadata,
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

func (c httpCourier) Deliver(
	ctx context.Context,
	endpoint string,
	peer yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) Receipt {
	req := yacyproto.TransferURLRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      peer,
		URLCount:    len(metadata),
		URLs:        metadata,
	}

	msg, err := c.exchange.Exchange(ctx, endpoint, yacyproto.PathTransferURL, req.Form())
	if err != nil {
		slog.WarnContext(
			ctx,
			"url metadata delivery failed",
			slog.String("peer", peer.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return Receipt{Outcome: Unreachable}
	}

	resp, err := yacyproto.ParseTransferURLResponse(msg)
	if err != nil {
		slog.WarnContext(
			ctx,
			"url metadata response not parsed",
			slog.String("peer", peer.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return Receipt{Outcome: Unreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK, "":
		return Receipt{Outcome: Accepted, URLsRejected: resp.ErrorURL}
	case yacyproto.ResultErrorNotGranted:
		return Receipt{Outcome: Deferred}
	case yacyproto.ResultWrongTarget:
		return Receipt{Outcome: Refused}
	default:
		slog.WarnContext(
			ctx,
			"url metadata delivery refused",
			slog.String("peer", peer.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}
