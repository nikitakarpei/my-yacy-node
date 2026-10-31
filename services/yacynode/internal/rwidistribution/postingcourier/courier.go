// Package postingcourier sends stored postings to a peer over the transferRWI
// protocol endpoint and reports how the peer responded and how long it took.
// An identified peer (signed seed) also gets the proofs that lead the postings
// to their postings roots.
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

type PostingsProofs interface {
	ProofsFor(
		ctx context.Context,
		postings []yacymodel.RWIPosting,
	) map[yacymodel.URLHash]yacymodel.PostingsProof
}

type Courier struct {
	formPoster     FormPoster
	peerRecords    PeerRecords
	postingsProofs PostingsProofs
	networkName    string
	self           yacymodel.Hash
	observer       Observer
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func New(
	formPoster FormPoster,
	peerRecords PeerRecords,
	postingsProofs PostingsProofs,
	networkName string,
	self yacymodel.Hash,
	observer Observer,
) Courier {
	return Courier{
		formPoster:     formPoster,
		peerRecords:    peerRecords,
		postingsProofs: postingsProofs,
		networkName:    networkName,
		self:           self,
		observer:       observer,
	}
}

func (c Courier) Offer(
	ctx context.Context,
	peer yacymodel.Hash,
	postings []yacymodel.RWIPosting,
) Receipt {
	record, _ := c.peerRecords.RecordOf(ctx, peer)
	baseURL, known := record.AnsweredBaseURL.Get()
	if !known {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without answered base URL",
			slog.String("peer", peer.String()),
		)

		return Receipt{Outcome: Unaddressable}
	}

	resp, answered := c.postTransferRWI(ctx, record.Callee, baseURL, postings)
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
		NetworkName:    c.networkName,
		Iam:            c.self,
		YouAre:         callee.Hash(),
		WordCount:      distinctWordCount(postings),
		EntryCount:     len(postings),
		Indexes:        postings,
		PostingsProofs: c.proofsForPostingsTo(ctx, postings, callee),
		Key:            yacyproto.TransferRWIKey(len(postings)),
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

func distinctWordCount(postings []yacymodel.RWIPosting) int {
	words := make(map[yacymodel.Hash]struct{}, len(postings))
	for _, posting := range postings {
		words[posting.WordHash] = struct{}{}
	}

	return len(words)
}

func (c Courier) proofsForPostingsTo(
	ctx context.Context,
	postings []yacymodel.RWIPosting,
	callee peerwire.Callee,
) map[yacymodel.URLHash]yacymodel.PostingsProof {
	if !callee.IsIdentified() {
		return nil
	}

	return c.postingsProofs.ProofsFor(ctx, postings)
}

func isTransferRWIReply(fields yacyproto.ResponseFields) bool {
	_, err := yacyproto.ParseTransferRWIResponse(fields)

	return err == nil
}
