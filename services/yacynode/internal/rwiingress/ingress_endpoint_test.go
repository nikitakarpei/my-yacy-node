package rwiingress

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
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
) (rwipostings.Receipt, error) {
	if r.busy {
		return rwipostings.Receipt{Busy: true, Pause: 5}, nil
	}
	r.received = append(r.received, postings...)

	return rwipostings.Receipt{}, nil
}

func endpointWith(intake *recordingIntake) transferRWIEndpoint {
	return transferRWIEndpoint{identity: localIdentity(), intake: intake}
}

func posting(word, urlSeed string) yacymodel.RWIPostingWireForm {
	return yacymodel.RWIPostingWireForm{
		WordHash: yacymodel.WordHash(word),
		Properties: map[string]string{
			yacymodel.ColURLHash:        yacymodel.WordHash(urlSeed).String(),
			yacymodel.ColLocalLinkCount: "1",
			yacymodel.ColHitCount:       "1",
		},
	}
}

func TestTransferRWIReportsBusy(t *testing.T) {
	endpoint := endpointWith(&recordingIntake{busy: true})

	req := yacyproto.TransferRWIRequest{
		NetworkName: "freeworld",
		YouAre:      localIdentity().Hash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPostingWireForm{posting("w1", "u1")},
	}

	resp, err := endpoint.Serve(context.Background(), req)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if resp.Result != yacyproto.ResultBusy {
		t.Fatalf("Result = %q, want busy", resp.Result)
	}
	if resp.Pause != 5 {
		t.Fatalf("Pause = %d, want 5", resp.Pause)
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
		Indexes:     []yacymodel.RWIPostingWireForm{posting("w1", "u1")},
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
