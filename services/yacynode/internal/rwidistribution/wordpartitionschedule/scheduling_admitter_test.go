package wordpartitionschedule_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type fakeAdmitter struct {
	admitted []yacymodel.RWIPosting
	err      error
}

func (f *fakeAdmitter) Admit(_ *vault.Txn, posting yacymodel.RWIPosting) error {
	if f.err != nil {
		return f.err
	}
	f.admitted = append(f.admitted, posting)

	return nil
}

func TestSchedulingAdmitterStoresPostingAndMakesItsWordPartitionDue(t *testing.T) {
	h := openHarness(t)
	wrapped := &fakeAdmitter{}
	admitter := h.schedule.AdmitterFrom(wrapped)
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")}

	h.write(t, func(tx *vault.Txn) error { return admitter.Admit(tx, posting) })

	if len(wrapped.admitted) != 1 || wrapped.admitted[0] != posting {
		t.Fatalf("wrapped admitter got %v, want [%v]", wrapped.admitted, posting)
	}
	want := yacymodel.WordPartitionOfPosting(posting, h.partitions)
	due := h.dueWordPartitions(t, 10)
	if len(due) != 1 || due[0] != want {
		t.Fatalf("due word partitions = %v, want [%v]", due, want)
	}
}

func TestSchedulingAdmitterReturnsWrappedErrorAndMakesNothingDue(t *testing.T) {
	h := openHarness(t)
	failure := errors.New("admit failed")
	admitter := h.schedule.AdmitterFrom(&fakeAdmitter{err: failure})

	err := h.vault.Update(t.Context(), func(tx *vault.Txn) error {
		return admitter.Admit(tx, yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")})
	})

	if !errors.Is(err, failure) {
		t.Fatalf("Admit error = %v, want %v", err, failure)
	}
	if due := h.dueWordPartitions(t, 10); len(due) != 0 {
		t.Fatalf("due word partitions = %v, want none", due)
	}
}
