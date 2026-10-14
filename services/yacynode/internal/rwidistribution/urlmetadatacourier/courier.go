// Package urlmetadatacourier delivers URL metadata to a peer over the
// transferURL protocol endpoint and reports how the peer responded and how long
// it took.
package urlmetadatacourier

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type Receipt struct {
	Outcome      Outcome
	URLsRejected []yacymodel.URLHash
}

type Observer interface {
	ObserveURLMetadataDeliveryAnswerTime(answerTime time.Duration)
}

type Courier struct {
	exchange    peerwire.FormExchange
	networkName string
	self        yacymodel.Hash
	now         func() time.Time
	observer    Observer
}

func New(
	exchange peerwire.FormExchange,
	networkName string,
	self yacymodel.Hash,
	now func() time.Time,
	observer Observer,
) Courier {
	return Courier{
		exchange:    exchange,
		networkName: networkName,
		self:        self,
		now:         now,
		observer:    observer,
	}
}

func (c Courier) Deliver(
	ctx context.Context,
	recipient yacymodel.Seed,
	metadata []yacymodel.URLMetadata,
) Receipt {
	peer := recipient.Hash
	address, addressable := recipient.NetworkAddress()
	if !addressable {
		slog.WarnContext(
			ctx,
			"url metadata not delivered to peer without network address",
			slog.String("peer", peer.String()),
		)

		return Receipt{Outcome: Unaddressable}
	}
	endpoint := address.String()

	req := yacyproto.TransferURLRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      peer,
		URLCount:    len(metadata),
		URLs:        metadata,
	}

	sentAt := c.now()
	msg, err := c.exchange.Exchange(ctx, endpoint, yacyproto.PathTransferURL, req.Form())
	c.observer.ObserveURLMetadataDeliveryAnswerTime(c.now().Sub(sentAt))
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
