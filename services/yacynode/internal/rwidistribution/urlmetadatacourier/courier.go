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

const deliveryBudget = 30 * time.Second

type Receipt struct {
	Outcome      Outcome
	URLsRejected []yacymodel.URLHash
}

type Observer interface {
	ObserveURLMetadataDeliveryAnswerTime(answerTime time.Duration)
}

type FormExchange interface {
	Exchange(
		ctx context.Context,
		request peerwire.Request,
		baseURLs []peerwire.BaseURL,
		accepts func(yacyproto.ResponseFields) bool,
		budget time.Duration,
	) (peerwire.Answer, bool)
}

type AnsweredBaseURLs interface {
	AnsweredBaseURLOf(ctx context.Context, peer yacymodel.Hash) (peerwire.BaseURL, bool)
}

type Courier struct {
	exchange         FormExchange
	answeredBaseURLs AnsweredBaseURLs
	networkName      string
	self             yacymodel.Hash
	observer         Observer
}

func New(
	exchange FormExchange,
	answeredBaseURLs AnsweredBaseURLs,
	networkName string,
	self yacymodel.Hash,
	observer Observer,
) Courier {
	return Courier{
		exchange:         exchange,
		answeredBaseURLs: answeredBaseURLs,
		networkName:      networkName,
		self:             self,
		observer:         observer,
	}
}

func (c Courier) Deliver(
	ctx context.Context,
	peer yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) Receipt {
	baseURL, known := c.answeredBaseURLs.AnsweredBaseURLOf(ctx, peer)
	if !known {
		slog.WarnContext(
			ctx,
			"url metadata not delivered to peer without answered base URL",
			slog.String("peer", peer.String()),
		)

		return Receipt{Outcome: Unaddressable}
	}

	req := yacyproto.TransferURLRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      peer,
		URLCount:    len(metadata),
		URLs:        metadata,
	}

	answer, answered := c.exchange.Exchange(
		ctx,
		peerwire.Request{Peer: peer, Path: yacyproto.PathTransferURL, Form: req.Form()},
		[]peerwire.BaseURL{baseURL},
		isTransferURLReply,
		deliveryBudget,
	)
	if !answered {
		slog.WarnContext(ctx, "url metadata delivery failed", slog.String("peer", peer.String()))

		return Receipt{Outcome: Unreachable}
	}
	c.observer.ObserveURLMetadataDeliveryAnswerTime(answer.Time)
	resp, _ := yacyproto.ParseTransferURLResponse(answer.Fields)

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
			slog.String("baseUrl", baseURL.String()),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}

func isTransferURLReply(fields yacyproto.ResponseFields) bool {
	_, err := yacyproto.ParseTransferURLResponse(fields)

	return err == nil
}
