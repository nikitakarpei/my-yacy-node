package urlmetadatacourier_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

func courierSeed(t testing.TB, server *httptest.Server) yacymodel.Seed {
	t.Helper()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host, portRaw, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	parsedHost, err := yacymodel.ParseHost(host)
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	port, err := yacymodel.ParsePort(portRaw)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	return yacymodel.Seed{
		Hash:           courierHash("peer"),
		PrimaryAddress: yacymodel.Some(parsedHost),
		Port:           yacymodel.Some(port),
	}
}

func urlResponder(resp yacyproto.TransferURLResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(resp.Encode().Encode()))
	}))
}

type steppingClock struct {
	at   time.Time
	step time.Duration
}

func (c *steppingClock) now() time.Time {
	c.at = c.at.Add(c.step)

	return c.at
}

type answerTimeRecorder struct{ answerTimes []time.Duration }

func (r *answerTimeRecorder) ObserveURLMetadataDeliveryAnswerTime(answerTime time.Duration) {
	r.answerTimes = append(r.answerTimes, answerTime)
}

func openURLMetadataCourierHarness(
	t *testing.T,
	server *httptest.Server,
) urlmetadatacourier.Courier {
	t.Helper()

	return openTimedURLMetadataCourierHarness(t, server, &answerTimeRecorder{})
}

func openTimedURLMetadataCourierHarness(
	t *testing.T,
	server *httptest.Server,
	recorder *answerTimeRecorder,
) urlmetadatacourier.Courier {
	t.Helper()

	return urlmetadatacourier.New(
		peerwire.NewFormExchange(server.Client()),
		"freeworld",
		courierHash("self"),
		(&steppingClock{at: time.Unix(1000, 0), step: time.Second}).now,
		recorder,
	)
}

func singleURLMetadata() []yacymodel.URLMetadata {
	return []yacymodel.URLMetadata{{Address: "http://example.com/u1"}}
}

func TestDeliverReportsAcceptedOnOK(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t, server), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
}

func TestDeliverReportsAcceptedWithRejectedURLs(t *testing.T) {
	rejected := urlHash("u1")
	server := urlResponder(yacyproto.TransferURLResponse{
		Result:   yacyproto.ResultOK,
		ErrorURL: []yacymodel.URLHash{rejected},
	})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t, server), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Accepted {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Accepted)
	}
	if len(receipt.URLsRejected) != 1 || receipt.URLsRejected[0] != rejected {
		t.Fatalf("URLsRejected = %v, want [%v]", receipt.URLsRejected, rejected)
	}
}

func TestDeliverReportsDeferredOnErrorNotGranted(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultErrorNotGranted})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t, server), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Deferred {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Deferred)
	}
}

func TestDeliverReportsRefusedOnWrongTarget(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultWrongTarget})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t, server), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Refused {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Refused)
	}
}

func TestDeliverReportsUnreachableOnTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), courierSeed(t, server), singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Unreachable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unreachable)
	}
}

func TestDeliverReportsUnaddressablePeer(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	defer server.Close()

	courier := openURLMetadataCourierHarness(t, server)

	receipt := courier.Deliver(
		context.Background(), yacymodel.Seed{Hash: courierHash("peer")}, singleURLMetadata(),
	)
	if receipt.Outcome != urlmetadatacourier.Unaddressable {
		t.Fatalf("outcome = %q, want %q", receipt.Outcome, urlmetadatacourier.Unaddressable)
	}
}

func TestDeliverReportsHowLongThePeerTookToAnswer(t *testing.T) {
	server := urlResponder(yacyproto.TransferURLResponse{Result: yacyproto.ResultOK})
	defer server.Close()
	recorder := &answerTimeRecorder{}

	openTimedURLMetadataCourierHarness(t, server, recorder).Deliver(
		context.Background(), courierSeed(t, server), singleURLMetadata(),
	)

	if len(recorder.answerTimes) != 1 || recorder.answerTimes[0] != time.Second {
		t.Fatalf("answer times = %v, want one of 1s", recorder.answerTimes)
	}
}
