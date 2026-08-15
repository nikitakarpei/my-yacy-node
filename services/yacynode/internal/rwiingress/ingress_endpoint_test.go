package rwiingress_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type stubRuntimeStatus struct{}

func (stubRuntimeStatus) Version(context.Context) string { return "1.0" }

func (stubRuntimeStatus) Uptime(context.Context) int { return 0 }

func localIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{Hash: yacymodel.WordHash("self"), NetworkName: "freeworld"}
}

type recordingIntake struct {
	received []yacymodel.RWIPosting
	busy     bool
}

func (r *recordingIntake) Receive(
	_ context.Context,
	postings []yacymodel.RWIPosting,
) (rwiadmission.Receipt, error) {
	if r.busy {
		return rwiadmission.Receipt{Busy: true, Pause: 5 * time.Second}, nil
	}
	r.received = append(r.received, postings...)

	return rwiadmission.Receipt{}, nil
}

func muxWith(intake *recordingIntake) *http.ServeMux {
	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(stubRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
	rwiingress.Mount(router, localIdentity(), intake)

	return mux
}

func posting(t *testing.T, word, urlSeed string) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHashFromWord(t, urlSeed),
		LocalLinks: 1,
		Hits:       1,
	}
}

func urlHashFromWord(t *testing.T, word string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(word).String())
	if err != nil {
		t.Fatalf("url hash for %q: %v", word, err)
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

	resp, err := yacyproto.ParseTransferRWIResponse(yacyproto.ParseMessage(string(body)))
	if err != nil {
		t.Fatalf("ParseTransferRWIResponse: %v", err)
	}

	return resp
}

func TestTransferRWIReportsBusy(t *testing.T) {
	mux := muxWith(&recordingIntake{busy: true})

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1", "u1")},
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
	intake := &recordingIntake{}
	mux := muxWith(intake)

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1", "u1")},
	}

	resp := transferRWI(t, mux, req)
	if resp.Result != yacyproto.TransferRWIResult(yacyproto.ResultOK) {
		t.Fatalf("Result = %q, want ok", resp.Result)
	}
	if len(intake.received) != 1 {
		t.Fatalf("received = %d postings, want 1", len(intake.received))
	}
}

func TestTransferRWIRejectsWrongNetwork(t *testing.T) {
	mux := muxWith(&recordingIntake{})

	req := yacyproto.TransferRWIRequest{NetworkName: "othernet", YouAre: localIdentity().Hash}

	resp := transferRWI(t, mux, req)
	if resp.Result != yacyproto.TransferRWIResult(yacyproto.ResultWrongTarget) {
		t.Fatalf("Result = %q, want wrong_target", resp.Result)
	}
}
