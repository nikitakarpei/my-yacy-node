package urlmetadatacourier_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peercontact"
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

func courierSeed(t testing.TB) yacymodel.Seed {
	t.Helper()

	host, err := yacymodel.ParseHost("192.0.2.1")
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}

	return yacymodel.Seed{
		Hash:           courierHash("peer"),
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(yacymodel.Port(8090)),
	}
}

type stubContacts struct {
	reply                 yacyproto.ResponseFields
	answers               bool
	withoutAnsweredOrigin bool
}

func answeringContacts(resp yacyproto.TransferURLResponse) *stubContacts {
	return &stubContacts{reply: resp.Encode(), answers: true}
}

func (s *stubContacts) Deliver(
	_ context.Context,
	_ yacymodel.PeerAddresses,
	question peercontact.Question,
	acceptsFromAnsweredOrigin func(yacyproto.ResponseFields) bool,
) (peerwire.Answer, bool) {
	check := acceptsFromAnsweredOrigin
	if s.withoutAnsweredOrigin {
		check = question.Accepts
	}
	if !s.answers || !check(s.reply) {
		return peerwire.Answer{}, false
	}

	return peerwire.Answer{Fields: s.reply, Time: time.Second}, true
}

type answerTimeRecorder struct{ answerTimes []time.Duration }

func (r *answerTimeRecorder) ObserveURLMetadataDeliveryAnswerTime(answerTime time.Duration) {
	r.answerTimes = append(r.answerTimes, answerTime)
}

func courierOver(contacts urlmetadatacourier.Contacts) urlmetadatacourier.Courier {
	return timedCourierOver(contacts, &answerTimeRecorder{})
}

func timedCourierOver(
	contacts urlmetadatacourier.Contacts,
	recorder *answerTimeRecorder,
) urlmetadatacourier.Courier {
	return urlmetadatacourier.New(contacts, "freeworld", courierHash("self"), recorder)
}

func singleURLMetadata() []yacymodel.URLMetadata {
	return []yacymodel.URLMetadata{{Address: "http://example.com/u1"}}
}

func TestDeliverReportsAcceptedOnOK(t *testing.T) {
	contacts := answeringContacts(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})

	courier := courierOver(contacts)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
}

func TestDeliverReportsAcceptedWithRejectedURLs(t *testing.T) {
	rejected := urlHash("u1")
	contacts := answeringContacts(yacyproto.TransferURLResponse{
		Result:   yacyproto.ResultOK,
		ErrorURL: []yacymodel.URLHash{rejected},
	})

	courier := courierOver(contacts)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
	if len(receipt.URLsRejected) != 1 || receipt.URLsRejected[0] != rejected {
		t.Fatalf("URLsRejected = %v, want [%v]", receipt.URLsRejected, rejected)
	}
}

func TestDeliverReportsDeferredOnErrorNotGranted(t *testing.T) {
	contacts := answeringContacts(
		yacyproto.TransferURLResponse{Result: yacyproto.ResultErrorNotGranted},
	)

	courier := courierOver(contacts)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Deferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Deferred)
	}
}

func TestDeliverReportsRefusedOnWrongTarget(t *testing.T) {
	contacts := answeringContacts(
		yacyproto.TransferURLResponse{Result: yacyproto.ResultWrongTarget},
	)

	courier := courierOver(contacts)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Refused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Refused)
	}
}

func TestDeliverReportsUnreachableOnTransportFailure(t *testing.T) {
	contacts := &stubContacts{}

	courier := courierOver(contacts)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Unreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unreachable)
	}
}

func TestDeliverReportsUnaddressablePeer(t *testing.T) {
	contacts := answeringContacts(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})

	courier := courierOver(contacts)

	receipt := courier.Deliver(
		context.Background(), yacymodel.Seed{Hash: courierHash("peer")}, singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Unaddressable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unaddressable)
	}
}

func TestDeliverReportsHowLongThePeerTookToAnswer(t *testing.T) {
	contacts := answeringContacts(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	recorder := &answerTimeRecorder{}

	timedCourierOver(contacts, recorder).Deliver(
		context.Background(), courierSeed(t), singleURLMetadata(),
	)

	if len(recorder.answerTimes) != 1 || recorder.answerTimes[0] != time.Second {
		t.Fatalf("answer times = %v, want one of 1s", recorder.answerTimes)
	}
}

func TestDeliverToAPeerWithoutAnsweredOriginCountsOnlyAnAcceptedDelivery(t *testing.T) {
	contacts := answeringContacts(
		yacyproto.TransferURLResponse{Result: yacyproto.ResultErrorNotGranted},
	)
	contacts.withoutAnsweredOrigin = true
	courier := courierOver(contacts)

	receipt := courier.Deliver(context.Background(), courierSeed(t), singleURLMetadata())

	if receipt.Outcome != urlmetadatacourier.Unreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unreachable)
	}
}
