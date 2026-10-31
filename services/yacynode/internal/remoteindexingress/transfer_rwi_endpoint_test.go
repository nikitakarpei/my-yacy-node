package remoteindexingress_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type stubRuntimeStatus struct{}

func (stubRuntimeStatus) Version(context.Context) string { return "1.0" }

func (stubRuntimeStatus) Uptime(context.Context) int { return 0 }

func localIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:         yacymodel.WordHash("self"),
		NetworkName:  "freeworld",
		Capabilities: yacymodel.PeerCapabilities{AcceptRemoteIndex: true},
	}
}

type recordingReceiver struct {
	received             []yacymodel.RWIPosting
	receivedPagePostings []yacymodel.PagePostings
	origins              []rwiorigin.Origin
	answer               remoteindexadmission.PostingReceipt
}

func (r *recordingReceiver) Receive(
	_ context.Context,
	origin rwiorigin.Origin,
	receivedPagePostings []yacymodel.PagePostings,
) (remoteindexadmission.PostingReceipt, error) {
	for _, pagePostings := range receivedPagePostings {
		r.received = append(r.received, pagePostings.Postings()...)
	}
	r.receivedPagePostings = append(r.receivedPagePostings, receivedPagePostings...)
	r.origins = append(r.origins, origin)

	return r.answer, nil
}

func muxWith(receiver *recordingReceiver) *http.ServeMux {
	return muxAs(localIdentity(), receiver)
}

type recordingURLReceiver struct {
	received []yacymodel.URLMetadata
	senders  []rwiorigin.Origin
}

func (r *recordingURLReceiver) Receive(
	_ context.Context,
	sender rwiorigin.Origin,
	metadata []yacymodel.URLMetadata,
) (remoteindexadmission.URLMetadataReceipt, error) {
	r.received = append(r.received, metadata...)
	r.senders = append(r.senders, sender)

	return remoteindexadmission.URLMetadataReceipt{}, nil
}

func muxAs(identity nodeidentity.Identity, receiver *recordingReceiver) *http.ServeMux {
	return mountedMux(identity, receiver, &recordingURLReceiver{})
}

func mountedMux(
	identity nodeidentity.Identity,
	postings *recordingReceiver,
	urlMetadata *recordingURLReceiver,
) *http.ServeMux {
	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(stubRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
	remoteindexingress.Mount(router, identity, postings, urlMetadata)

	return mux
}

func posting(t *testing.T, word string) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    pageURLHash(t),
		Language:   yacymodel.LanguageOfUndeclaredDocument,
		LocalLinks: 1,
		Hits:       1,
	}
}

func pageURLHash(t *testing.T) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash("page").String())
	if err != nil {
		t.Fatalf("page url hash: %v", err)
	}

	return hash
}

func transferRWI(
	t *testing.T,
	mux *http.ServeMux,
	req yacyproto.TransferRWIRequest,
) yacyproto.TransferRWIResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathTransferRWI,
		strings.NewReader(req.Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %q", rec.Code, rec.Body.String())
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	resp, err := yacyproto.ParseTransferRWIResponse(yacyproto.ParseResponseFields(body))
	if err != nil {
		t.Fatalf("ParseTransferRWIResponse: %v", err)
	}

	return resp
}

func TestTransferRWIReportsBusy(t *testing.T) {
	mux := muxWith(&recordingReceiver{
		answer: remoteindexadmission.PostingReceipt{Busy: true, Pause: 5 * time.Second},
	})

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1")},
	}

	resp := transferRWI(t, mux, req)
	if resp.Result != yacyproto.ResultBusy {
		t.Fatalf("Result = %q, want busy", resp.Result)
	}
	if resp.Pause != 5*time.Second {
		t.Fatalf("Pause = %v, want 5s", resp.Pause)
	}
}

func TestTransferRWIStoresAndAnswers(t *testing.T) {
	receiver := &recordingReceiver{}
	mux := muxWith(receiver)
	sendingPeer := yacymodel.WordHash("sending peer")

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		Iam:         sendingPeer,
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1")},
	}

	resp := transferRWI(t, mux, req)
	if resp.Result != yacyproto.TransferRWIResult(yacyproto.ResultOK) {
		t.Fatalf("Result = %q, want ok", resp.Result)
	}
	if len(receiver.received) != 1 {
		t.Fatalf("received = %d postings, want 1", len(receiver.received))
	}
	if want := rwiorigin.Of("192.0.2.1"); receiver.origins[0] != want {
		t.Fatalf("origin = %+v, want %+v", receiver.origins[0], want)
	}
}

func TestTransferRWIHandsThePostingsProofsToTheReceiver(t *testing.T) {
	receiver := &recordingReceiver{}
	mux := muxWith(receiver)
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(2)
	if err != nil {
		t.Fatal(err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{posting(t, "w1"), posting(t, "w2")},
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		t.Fatal(err)
	}
	sent, _ := crawled.NarrowedTo(crawled.Postings()[:1]).Get()
	proof, _ := sent.Proof().Get()

	transferRWI(t, mux, yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     sent.Postings(),
		PostingsProofs: map[yacymodel.URLHash]yacymodel.PostingsProof{
			pageURLHash(t): proof,
		},
	})

	if len(receiver.receivedPagePostings) != 1 || !crawled.Root().Present() ||
		receiver.receivedPagePostings[0].Root() != crawled.Root() {
		t.Fatalf(
			"received page postings = %+v, want the root %+v",
			receiver.receivedPagePostings,
			crawled.Root(),
		)
	}
}

func TestTransferRWIHandsOverThePostingsOfAURLWithoutAProofBesideNoPathOfThem(
	t *testing.T,
) {
	receiver := &recordingReceiver{}
	mux := muxWith(receiver)
	junkProof, err := yacymodel.ParsePostingsProofBytes(
		append([]byte{1, 2, 0}, make([]byte, 32)...),
	)
	if err != nil {
		t.Fatal(err)
	}

	transferRWI(t, mux, yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   2,
		EntryCount:  2,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1"), posting(t, "w2")},
		PostingsProofs: map[yacymodel.URLHash]yacymodel.PostingsProof{
			pageURLHash(t): junkProof,
		},
	})

	if len(receiver.receivedPagePostings) != 1 ||
		receiver.receivedPagePostings[0].AmountOfPostings() != 2 ||
		receiver.receivedPagePostings[0].Proof().Present() {
		t.Fatalf(
			"received page postings = %+v, want both postings together without proof",
			receiver.receivedPagePostings,
		)
	}
}

func TestTransferRWIHandsOverThePostingsOfEachURLTogether(t *testing.T) {
	receiver := &recordingReceiver{}
	mux := muxWith(receiver)
	onOtherPage := posting(t, "w1")
	otherURL, err := yacymodel.ParseURLHash(yacymodel.WordHash("other page").String())
	if err != nil {
		t.Fatal(err)
	}
	onOtherPage.URLHash = otherURL

	transferRWI(t, mux, yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   2,
		EntryCount:  3,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1"), onOtherPage, posting(t, "w2")},
	})

	if len(receiver.receivedPagePostings) != 2 {
		t.Fatalf("page postings = %d, want one for each url", len(receiver.receivedPagePostings))
	}
	if got := receiver.receivedPagePostings[0]; got.URL() != pageURLHash(t) ||
		got.AmountOfPostings() != 2 {
		t.Errorf("first page postings = %+v, want both postings of the page", got)
	}
}

func TestTransferRWIHandsOverTheFirstPostingOfARepeatedWordWithTheProofOfItsURL(t *testing.T) {
	receiver := &recordingReceiver{}
	mux := muxWith(receiver)
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(2)
	if err != nil {
		t.Fatal(err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{posting(t, "w1"), posting(t, "w2")},
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		t.Fatal(err)
	}
	proof, _ := crawled.Proof().Get()
	repeated := posting(t, "w1")
	repeated.Hits = 2

	transferRWI(t, mux, yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   2,
		EntryCount:  3,
		Indexes:     append(crawled.Postings(), repeated),
		PostingsProofs: map[yacymodel.URLHash]yacymodel.PostingsProof{
			pageURLHash(t): proof,
		},
	})

	if len(receiver.receivedPagePostings) != 1 ||
		!slices.Equal(receiver.receivedPagePostings[0].Postings(), crawled.Postings()) ||
		receiver.receivedPagePostings[0].Root() != crawled.Root() {
		t.Fatalf("page postings = %+v, want the first posting of each word under the root",
			receiver.receivedPagePostings)
	}
}

func TestTransferRWIRejectsWrongNetwork(t *testing.T) {
	mux := muxWith(&recordingReceiver{})

	req := yacyproto.TransferRWIRequest{NetworkName: "othernet", YouAre: localIdentity().Hash}

	resp := transferRWI(t, mux, req)
	if resp.Result != yacyproto.TransferRWIResult(yacyproto.ResultWrongTarget) {
		t.Fatalf("Result = %q, want wrong_target", resp.Result)
	}
}

func TestTransferRWIRefusesWhenRemoteIndexIsNotAccepted(t *testing.T) {
	identity := localIdentity()
	identity.Capabilities.AcceptRemoteIndex = false
	receiver := &recordingReceiver{}
	mux := muxAs(identity, receiver)

	resp := transferRWI(t, mux, yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      identity.Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1")},
	})

	if resp.Result != yacyproto.ResultNotGranted {
		t.Fatalf("Result = %q, want not granted", resp.Result)
	}
	if len(receiver.received) != 0 {
		t.Fatalf("received = %d postings, want none reach the receiver", len(receiver.received))
	}
}
