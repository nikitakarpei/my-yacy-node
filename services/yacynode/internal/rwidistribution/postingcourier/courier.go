// Package postingcourier sends stored postings to a peer over the transferRWI
// protocol endpoint and reports how the peer responded and how long it took.
package postingcourier

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const offerBudget = 30 * time.Second

type Receipt struct {
	Outcome           Outcome
	RequestedPause    time.Duration
	URLsUnknownToPeer []yacymodel.URLHash
	URLsRejected      []yacymodel.URLHash
}

type Observer interface {
	ObservePostingOfferAnswerTime(answerTime time.Duration)
}

type FormPoster interface {
	TryAt(
		ctx context.Context,
		request peerwire.Request,
		baseURL peerwire.BaseURL,
		accepts func(yacyproto.ResponseFields) bool,
		budget time.Duration,
	) (peerwire.Answer, bool)
}

type PeerRecords interface {
	RecordOf(ctx context.Context, peer yacymodel.Hash) (peerroster.PeerRecord, bool)
}

type Courier struct {
	formPoster  FormPoster
	peerRecords PeerRecords
	networkName string
	self        yacymodel.Hash
	observer    Observer
}

func New(
	formPoster FormPoster,
	peerRecords PeerRecords,
	networkName string,
	self yacymodel.Hash,
	observer Observer,
) Courier {
	return Courier{
		formPoster:  formPoster,
		peerRecords: peerRecords,
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
	record, _ := c.peerRecords.RecordOf(ctx, peer)
	baseURL, answered := record.AnsweredBaseURL.Get()
	if !answered {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without answered base URL",
			slog.String("peer", peer.String()),
		)

		return Receipt{Outcome: Unaddressable}
	}

	resp, replied := c.postTransferRWI(ctx, record.Callee, baseURL, postings)
	if !replied {
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
			slog.String("baseUrl", baseURL.String()),
		)

		return Receipt{Outcome: Overloaded, RequestedPause: resp.Pause}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", peer.String()),
			slog.String("baseUrl", baseURL.String()),
			slog.String("result", string(resp.Result)),
		)

		return Receipt{Outcome: Refused}
	}
}

func (c Courier) postTransferRWI(
	ctx context.Context,
	callee peerwire.Callee,
	baseURL peerwire.BaseURL,
	postings []yacymodel.RWIPosting,
) (yacyproto.TransferRWIResponse, bool) {
	req := yacyproto.TransferRWIRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      callee.Hash(),
		WordCount:   distinctWordCount(postings),
		EntryCount:  len(postings),
		Indexes:     postings,
		Key:         yacyproto.TransferRWIKey(len(postings)),
	}

	answer, answered := c.formPoster.TryAt(
		ctx,
		peerwire.Request{
			Callee: callee,
			Path:   yacyproto.PathTransferRWI,
			Form:   req.Form(),
		},
		baseURL,
		isTransferRWIReply,
		offerBudget,
	)
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
