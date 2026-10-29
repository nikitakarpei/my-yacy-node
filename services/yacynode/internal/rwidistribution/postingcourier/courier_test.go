package postingcourier_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const courierHashFiller = "AAAAAAAAAAAA"

func courierHash(base string) yacymodel.Hash {
	padded := base + courierHashFiller
	hash, err := yacymodel.ParseHash(padded[:yacymodel.HashLength])
	if err != nil {
		panic(err)
	}

	return hash
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

type stubPeerWire struct {
	reply                  yacyproto.ResponseFields
	answers                bool
	withoutAnsweredBaseURL bool
	identity               yacymodel.Optional[yacymodel.PeerIdentity]
	requests               []peerwire.Request
}

func answeringPeerWire(resp yacyproto.TransferRWIResponse) *stubPeerWire {
	return &stubPeerWire{reply: resp.Encode(), answers: true}
}

func (s *stubPeerWire) RecordOf(context.Context, yacymodel.Hash) (peerroster.PeerRecord, bool) {
	record := peerroster.PeerRecord{Identity: s.identity}
	if !s.withoutAnsweredBaseURL {
		record.AnsweredBaseURL = yacymodel.Some(peerwire.BaseURL{Scheme: peerwire.HTTP})
	}

	return record, true
}

func (s *stubPeerWire) TryAt(
	_ context.Context,
	request peerwire.Request,
	_ peerwire.BaseURL,
	accepts func(yacyproto.ResponseFields) bool,
	_ time.Duration,
) (peerwire.Answer, bool) {
	s.requests = append(s.requests, request)
	if !s.answers || !accepts(s.reply) {
		return peerwire.Answer{}, false
	}

	return peerwire.Answer{Fields: s.reply, Time: time.Second}, true
}

type answerTimeRecorder struct{ answerTimes []time.Duration }

func (r *answerTimeRecorder) ObservePostingOfferAnswerTime(answerTime time.Duration) {
	r.answerTimes = append(r.answerTimes, answerTime)
}

func courierOver(wire *stubPeerWire) postingcourier.Courier {
	return timedCourierOver(wire, &answerTimeRecorder{})
}

func timedCourierOver(
	wire *stubPeerWire,
	recorder *answerTimeRecorder,
) postingcourier.Courier {
	return postingcourier.New(wire, wire, "freeworld", courierHash("self"), recorder)
}

func singlePosting() []yacymodel.RWIPosting {
	return []yacymodel.RWIPosting{fakePosting(yacymodel.WordHash("w1"), urlHash("u1"))}
}

func TestOfferReportsAcceptedOnOK(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK})

	courier := courierOver(wire)

	receipt := courier.Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)
	if receipt.Outcome != postingcourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Accepted)
	}
}

func TestOfferNamesTheBoundIdentityOfThePeer(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK})
	wire.identity = yacymodel.Some(peerkeytest.Key(0).Identity())

	courierOver(wire).Offer(context.Background(), courierHash("peer"), singlePosting())

	if len(wire.requests) != 1 || wire.requests[0].Identity != wire.identity {
		t.Fatalf("requests = %v, want one naming %v", wire.requests, wire.identity)
	}
}

func TestOfferReturnsUnknownURLsWithoutActingOnThem(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	unknown := fakePosting(word, url).URLHash
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{
		Result:     yacyproto.ResultOK,
		UnknownURL: []yacymodel.URLHash{unknown},
	})

	courier := courierOver(wire)

	receipt := courier.Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)
	if receipt.Outcome != postingcourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Accepted)
	}
	if len(receipt.URLsUnknownToPeer) != 1 || receipt.URLsUnknownToPeer[0] != unknown {
		t.Fatalf("URLsUnknownToPeer = %v, want [%v]", receipt.URLsUnknownToPeer, unknown)
	}
}

func TestOfferReturnsURLsThePeerRejected(t *testing.T) {
	rejected := urlHash("u1")
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{
		Result:   yacyproto.ResultOK,
		ErrorURL: []yacymodel.URLHash{rejected},
	})

	receipt := courierOver(wire).Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)

	if len(receipt.URLsRejected) != 1 || receipt.URLsRejected[0] != rejected {
		t.Fatalf("URLsRejected = %v, want [%v]", receipt.URLsRejected, rejected)
	}
}

func TestOfferReportsDeferredWithPeerPause(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{
		Result: yacyproto.ResultBusy,
		Pause:  30 * time.Second,
	})

	courier := courierOver(wire)

	receipt := courier.Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)
	if receipt.Outcome != postingcourier.Deferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Deferred)
	}
	if receipt.RequestedPause != 30*time.Second {
		t.Fatalf("RequestedPause = %v, want 30s", receipt.RequestedPause)
	}
}

func TestOfferReportsOverloadedPeer(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{
		Result: yacyproto.ResultTooHighLoad,
		Pause:  time.Minute,
	})

	courier := courierOver(wire)

	receipt := courier.Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)
	if receipt.Outcome != postingcourier.Overloaded {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Overloaded)
	}
	if receipt.RequestedPause != time.Minute {
		t.Fatalf("RequestedPause = %v, want 1m", receipt.RequestedPause)
	}
}

func TestOfferReportsRefusalOnUnexpectedResult(t *testing.T) {
	wire := answeringPeerWire(
		yacyproto.TransferRWIResponse{Result: yacyproto.ResultWrongTarget},
	)

	courier := courierOver(wire)

	receipt := courier.Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)
	if receipt.Outcome != postingcourier.Refused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Refused)
	}
}

func TestOfferReportsUnreachablePeerOnTransportFailure(t *testing.T) {
	wire := &stubPeerWire{}

	courier := courierOver(wire)

	receipt := courier.Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)
	if receipt.Outcome != postingcourier.Unreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Unreachable)
	}
}

func TestOfferReportsUnaddressableForAPeerWithoutAnsweredBaseURL(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK})
	wire.withoutAnsweredBaseURL = true

	receipt := courierOver(
		wire,
	).Offer(context.Background(), courierHash("peer"), singlePosting())
	if receipt.Outcome != postingcourier.Unaddressable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, postingcourier.Unaddressable)
	}
}

func TestOfferReportsHowLongThePeerTookToAnswer(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK})
	recorder := &answerTimeRecorder{}

	timedCourierOver(wire, recorder).Offer(
		context.Background(),
		courierHash("peer"),
		singlePosting(),
	)

	if len(recorder.answerTimes) != 1 || recorder.answerTimes[0] != time.Second {
		t.Fatalf("answer times = %v, want one of 1s", recorder.answerTimes)
	}
}
