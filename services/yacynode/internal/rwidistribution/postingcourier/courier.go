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
	AnsweredOriginOf(peer yacymodel.Hash) (peerwire.Origin, bool)
	Deliver(
		ctx context.Context,
		origin peerwire.Origin,
		question peercontact.Question,
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
	peer yacymodel.Hash,
	postings []yacymodel.RWIPosting,
) Receipt {
	origin, known := c.contacts.AnsweredOriginOf(peer)
	if !known {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without answered origin",
			slog.String("peer", peer.String()),
		)

		return Receipt{Outcome: Unaddressable}
	}

	resp, answered := c.postTransferRWI(ctx, origin, peer, postings)
	if !answered {
		slog.WarnContext(ctx, "posting offer failed", slog.String("peer", peer.String()))

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
			slog.String("peer", peer.String()),
			slog.String("origin", origin.String()),
		)

		return Receipt{Outcome: Overloaded, RequestedPause: resp.Pause}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", peer.String()),
			slog.String("origin", origin.String()),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}

func (c Courier) postTransferRWI(
	ctx context.Context,
	origin peerwire.Origin,
	peer yacymodel.Hash,
	postings []yacymodel.RWIPosting,
) (yacyproto.TransferRWIResponse, bool) {
	req := yacyproto.TransferRWIRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      peer,
		WordCount:   distinctWordCount(postings),
		EntryCount:  len(postings),
		Indexes:     postings,
		Key:         yacyproto.TransferRWIKey(len(postings)),
	}

	answer, answered := c.contacts.Deliver(ctx, origin, peercontact.Question{
		Request: peerwire.Request{Peer: peer, Path: yacyproto.PathTransferRWI, Form: req.Form()},
		Accepts: isTransferRWIReply,
	})
	if !answered {
		return yacyproto.TransferRWIResponse{}, false
	}
	c.observer.ObservePostingOfferAnswerTime(answer.Time)
	resp, _ := yacyproto.ParseTransferRWIResponse(answer.Fields)

	return resp, true
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
