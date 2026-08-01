package rwidistribution

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type urlMetadataReceipt struct {
	Outcome      urlMetadataOutcome
	URLsRejected []yacymodel.URLHash
}

type httpURLMetadataCourier struct {
	exchange    peerwire.MessageExchange
	networkName string
	self        yacymodel.Hash
}

func (c httpURLMetadataCourier) Deliver(
	ctx context.Context,
	endpoint string,
	peer yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) urlMetadataReceipt {
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

		return urlMetadataReceipt{Outcome: urlMetadataUnreachable}
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

		return urlMetadataReceipt{Outcome: urlMetadataUnreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK, "":
		return urlMetadataReceipt{Outcome: urlMetadataAccepted, URLsRejected: resp.ErrorURL}
	case yacyproto.ResultErrorNotGranted:
		return urlMetadataReceipt{Outcome: urlMetadataDeferred}
	case yacyproto.ResultWrongTarget:
		return urlMetadataReceipt{Outcome: urlMetadataRefused}
	default:
		slog.WarnContext(
			ctx,
			"url metadata delivery refused",
			slog.String("peer", peer.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)

		return urlMetadataReceipt{Outcome: urlMetadataRefused}
	}
}
