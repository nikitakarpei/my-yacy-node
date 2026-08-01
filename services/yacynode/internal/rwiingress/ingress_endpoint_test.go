package rwiingress

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

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

func endpointWith(intake *recordingIntake) transferRWIEndpoint {
	return transferRWIEndpoint{identity: localIdentity(), intake: intake}
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

func TestTransferRWIReportsBusy(t *testing.T) {
	endpoint := endpointWith(&recordingIntake{busy: true})

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1", "u1")},
	}

	resp, err := endpoint.Serve(context.Background(), req)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Result != yacyproto.ResultBusy {
		t.Fatalf("Result = %q, want busy", resp.Result)
	}
	if resp.Pause != 5*time.Second {
		t.Fatalf("Pause = %v, want 5s", resp.Pause)
	}
}

func TestTransferRWIStoresAndAnswers(t *testing.T) {
	intake := &recordingIntake{}
	endpoint := endpointWith(intake)

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{posting(t, "w1", "u1")},
	}

	resp, err := endpoint.Serve(context.Background(), req)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Result != yacyproto.TransferRWIResult(yacyproto.ResultOK) {
		t.Fatalf("Result = %q, want ok", resp.Result)
	}
	if len(intake.received) != 1 {
		t.Fatalf("received = %d postings, want 1", len(intake.received))
	}
}

func TestTransferRWIRejectsWrongNetwork(t *testing.T) {
	endpoint := endpointWith(&recordingIntake{})

	req := yacyproto.TransferRWIRequest{NetworkName: "othernet", YouAre: localIdentity().Hash}

	resp, err := endpoint.Serve(context.Background(), req)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Result != yacyproto.TransferRWIResult(yacyproto.ResultWrongTarget) {
		t.Fatalf("Result = %q, want wrong_target", resp.Result)
	}
}
