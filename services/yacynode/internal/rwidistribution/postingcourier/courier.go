// Package postingcourier sends stored postings to a peer over the transferRWI
// protocol endpoint and reports how the peer responded and how long it took.
package postingcourier

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
	Outcome           Outcome
	RequestedPause    time.Duration
	URLsUnknownToPeer []yacymodel.URLHash
	URLsRejected      []yacymodel.URLHash
}

type Observer interface {
	ObservePostingOfferAnswerTime(answerTime time.Duration)
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

func (c Courier) Offer(
	ctx context.Context,
	recipient yacymodel.Seed,
	postings []yacymodel.RWIPosting,
) Receipt {
	addresses, addressable := recipient.Addresses()
	if !addressable {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without addresses",
			slog.String("peer", recipient.Hash.String()),
		)

		return Receipt{Outcome: Unaddressable}
	}

	resp, origin, answered := c.postTransferRWI(ctx, addresses, recipient.Hash, postings)
	if !answered {
		slog.WarnContext(ctx, "posting offer failed", slog.String("peer", recipient.Hash.String()))

		return Receipt{Outcome: Unreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK:
		return Receipt{
			Outcome:           Accepted,
			URLsUnknownToPeer: resp.UnknownURL,
			URLsRejected:      resp.ErrorURL,
		}
	case yacyproto.ResultBusy, yacyproto.ResultNotGranted:
		return Receipt{Outcome: Deferred, RequestedPause: resp.Pause}
	case yacyproto.ResultTooHighLoad:
		slog.WarnContext(
			ctx,
			"posting offer rejected by loaded peer",
			slog.String("peer", recipient.Hash.String()),
			slog.String("origin", origin.String()),
		)

		return Receipt{Outcome: Overloaded, RequestedPause: resp.Pause}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", recipient.Hash.String()),
			slog.String("origin", origin.String()),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}

func (c Courier) postTransferRWI(
	ctx context.Context,
	addresses yacymodel.PeerAddresses,
	peer yacymodel.Hash,
	postings []yacymodel.RWIPosting,
) (yacyproto.TransferRWIResponse, peerwire.Origin, bool) {
	req := yacyproto.TransferRWIRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      peer,
		WordCount:   distinctWordCount(postings),
		EntryCount:  len(postings),
		Indexes:     postings,
		Key:         yacyproto.TransferRWIKey(len(postings)),
	}

	answer, answered := c.contacts.Deliver(ctx, addresses, peercontact.Question{
		Request: peerwire.Request{Peer: peer, Path: yacyproto.PathTransferRWI, Form: req.Form()},
		Accepts: isAcceptedTransferRWIReply,
	}, isTransferRWIReply)
	if !answered {
		return yacyproto.TransferRWIResponse{}, peerwire.Origin{}, false
	}
	c.observer.ObservePostingOfferAnswerTime(answer.Time)
	resp, _ := yacyproto.ParseTransferRWIResponse(answer.Fields)

	return resp, answer.Origin, true
}

func isAcceptedTransferRWIReply(fields yacyproto.ResponseFields) bool {
	resp, err := yacyproto.ParseTransferRWIResponse(fields)

	return err == nil && resp.Result == yacyproto.ResultOK
}

func isTransferRWIReply(fields yacyproto.ResponseFields) bool {
	_, err := yacyproto.ParseTransferRWIResponse(fields)

	return err == nil
}

func distinctWordCount(postings []yacymodel.RWIPosting) int {
	words := make(map[yacymodel.Hash]struct{}, len(postings))
	for _, posting := range postings {
		words[posting.WordHash] = struct{}{}
	}

	return len(words)
}
