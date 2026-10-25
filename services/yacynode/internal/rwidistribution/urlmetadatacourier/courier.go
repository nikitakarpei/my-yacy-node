// Package urlmetadatacourier delivers URL metadata to a peer over the
// transferURL protocol endpoint and reports how the peer responded and how long
// it took.
package urlmetadatacourier

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peercontact"
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

type Contacts interface {
	Deliver(
		ctx context.Context,
		addresses yacymodel.PeerAddresses,
		question peercontact.Question,
		accepts func(yacyproto.ResponseFields) bool,
	) (peerwire.Answer, bool)
}

type Courier struct {
	contacts    Contacts
	networkName string
	self        yacymodel.Hash
	observer    Observer
}

func New(
	contacts Contacts,
	networkName string,
	self yacymodel.Hash,
	observer Observer,
) Courier {
	return Courier{
		contacts:    contacts,
		networkName: networkName,
		self:        self,
		observer:    observer,
	}
}

func (c Courier) Deliver(
	ctx context.Context,
	recipient yacymodel.Seed,
	metadata []yacymodel.URLMetadata,
) Receipt {
	peer := recipient.Hash
	addresses, addressable := recipient.Addresses()
	if !addressable {
		slog.WarnContext(
			ctx,
			"url metadata not delivered to peer without addresses",
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

	answer, answered := c.contacts.Deliver(ctx, addresses, peercontact.Question{
		Request: peerwire.Request{Peer: peer, Path: yacyproto.PathTransferURL, Form: req.Form()},
		Accepts: isAcceptedTransferURLReply,
	}, isTransferURLReply)
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
			slog.String("origin", answer.Origin.String()),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}

func isAcceptedTransferURLReply(fields yacyproto.ResponseFields) bool {
	resp, err := yacyproto.ParseTransferURLResponse(fields)

	return err == nil && resp.Result == yacyproto.ResultOK
}

func isTransferURLReply(fields yacyproto.ResponseFields) bool {
	_, err := yacyproto.ParseTransferURLResponse(fields)

	return err == nil
}
