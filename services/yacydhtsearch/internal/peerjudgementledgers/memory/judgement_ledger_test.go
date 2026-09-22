package memory_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func honoredJudgementOf(peer string) peerjudgements.RecordedJudgement {
	return peerjudgements.RecordedJudgement{
		Form: peerjudgements.NamedDocuments,
		JudgedPeer: peerjudgements.JudgedPeer{
			PeerAtVersion: peerjudgements.PeerAtVersion{
				Peer:    yacymodel.WordHash(peer),
				Version: "yacy_v1.925",
			},
			Judgement: peerjudgements.Honored,
		},
		JudgedAt: time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC),
	}
}

func TestAHeldJudgementIsGivenBackForItsFormAndPeer(t *testing.T) {
	t.Parallel()

	ledger := memory.New(2)
	judgement := honoredJudgementOf("one")

	ledger.HoldJudgement(t.Context(), judgement)

	heldJudgement, held := ledger.JudgementOf(
		t.Context(), peerjudgements.NamedDocuments, judgement.Peer,
	).Get()
	if !held || heldJudgement != judgement {
		t.Fatalf("JudgementOf = %+v, %v, want the judgement held", heldJudgement, held)
	}
}

func TestTheJudgementOfAPeerBeyondTheCapacityIsDropped(t *testing.T) {
	t.Parallel()

	ledger := memory.New(1)
	droppedJudgement := honoredJudgementOf("one")

	ledger.HoldJudgement(t.Context(), droppedJudgement)
	ledger.HoldJudgement(t.Context(), honoredJudgementOf("two"))

	held := ledger.JudgementOf(
		t.Context(), peerjudgements.NamedDocuments, droppedJudgement.Peer,
	).Present()
	if held {
		t.Fatalf("JudgementOf gave back the judgement of %+v, want none", droppedJudgement.Peer)
	}
}
