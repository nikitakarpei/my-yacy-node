package urlmetadatacourier_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
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

type stubPeerWire struct {
	reply                  yacyproto.ResponseFields
	answers                bool
	withoutAnsweredBaseURL bool
	identity               yacymodel.Optional[yacymodel.PeerIdentity]
	requests               []peerwire.Request
}

func answeringPeerWire(resp yacyproto.TransferURLResponse) *stubPeerWire {
	return &stubPeerWire{reply: resp.Encode(), answers: true}
}

func (s *stubPeerWire) RecordOf(
	_ context.Context,
	peer yacymodel.Hash,
) (peerroster.PeerRecord, bool) {
	record := peerroster.PeerRecord{Callee: peerwire.UnidentifiedCallee(peer)}
	if identity, identified := s.identity.Get(); identified {
		record.Callee = peerwire.IdentifiedCallee(identity)
	}
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

func (r *answerTimeRecorder) ObserveURLMetadataDeliveryAnswerTime(answerTime time.Duration) {
	r.answerTimes = append(r.answerTimes, answerTime)
}

func courierOver(wire *stubPeerWire) urlmetadatacourier.Courier {
	return timedCourierOver(wire, &answerTimeRecorder{})
}

func timedCourierOver(
	wire *stubPeerWire,
	recorder *answerTimeRecorder,
) urlmetadatacourier.Courier {
	return urlmetadatacourier.New(wire, wire, "freeworld", courierHash("self"), recorder)
}

func singleURLMetadata() []yacymodel.URLMetadata {
	return []yacymodel.URLMetadata{{Address: "http://example.com/u1"}}
}

func TestDeliverReportsAcceptedOnOK(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})

	courier := courierOver(wire)

	receipt := courier.Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
}

func TestDeliverNamesTheBoundIdentityOfThePeer(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	wire.identity = yacymodel.Some(peerkeytest.Key(0).Identity())

	courierOver(wire).Deliver(context.Background(), courierHash("peer"), singleURLMetadata())

	if len(wire.requests) != 1 || wire.requests[0].Callee.Identity() != wire.identity {
		t.Fatalf("requests = %v, want one naming %v", wire.requests, wire.identity)
	}
}

func TestDeliverReportsAcceptedWithRejectedURLs(t *testing.T) {
	rejected := urlHash("u1")
	wire := answeringPeerWire(yacyproto.TransferURLResponse{
		Result:   yacyproto.ResultOK,
		ErrorURL: []yacymodel.URLHash{rejected},
	})

	courier := courierOver(wire)

	receipt := courier.Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
	if len(receipt.URLsRejected) != 1 || receipt.URLsRejected[0] != rejected {
		t.Fatalf("URLsRejected = %v, want [%v]", receipt.URLsRejected, rejected)
	}
}

func TestDeliverReportsDeferredOnErrorNotGranted(t *testing.T) {
	wire := answeringPeerWire(
		yacyproto.TransferURLResponse{Result: yacyproto.ResultErrorNotGranted},
	)

	courier := courierOver(wire)

	receipt := courier.Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Deferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Deferred)
	}
}

func TestDeliverReportsRefusedOnWrongTarget(t *testing.T) {
	wire := answeringPeerWire(
		yacyproto.TransferURLResponse{Result: yacyproto.ResultWrongTarget},
	)

	courier := courierOver(wire)

	receipt := courier.Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Refused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Refused)
	}
}

func TestDeliverReportsUnreachableOnTransportFailure(t *testing.T) {
	wire := &stubPeerWire{}

	courier := courierOver(wire)

	receipt := courier.Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Unreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unreachable)
	}
}

func TestDeliverReportsUnaddressableForAPeerWithoutAnsweredBaseURL(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	wire.withoutAnsweredBaseURL = true

	receipt := courierOver(wire).Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Unaddressable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unaddressable)
	}
}

func TestDeliverReportsHowLongThePeerTookToAnswer(t *testing.T) {
	wire := answeringPeerWire(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	recorder := &answerTimeRecorder{}

	timedCourierOver(wire, recorder).Deliver(
		context.Background(), courierHash("peer"), singleURLMetadata(),
	)

	if len(recorder.answerTimes) != 1 || recorder.answerTimes[0] != time.Second {
		t.Fatalf("answer times = %v, want one of 1s", recorder.answerTimes)
	}
}
