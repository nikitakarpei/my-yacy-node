package wordpartitionschedule_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
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
	admitter := h.schedule.AdmitterFrom(wrapped, wordpartitionschedule.DiscardDueObserver)
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
	admitter := h.schedule.AdmitterFrom(
		&fakeAdmitter{err: failure},
		wordpartitionschedule.DiscardDueObserver,
	)

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

type recordingDueObserver struct {
	positions []yacymodel.DHTRingPosition
}

func (r *recordingDueObserver) ObserveWordPartitionDue(position yacymodel.DHTRingPosition) {
	r.positions = append(r.positions, position)
}

func TestSchedulingAdmitterReportsTheDuePositionOnlyAfterCommit(t *testing.T) {
	h := openHarness(t)
	observer := &recordingDueObserver{}
	admitter := h.schedule.AdmitterFrom(&fakeAdmitter{}, observer)
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")}
	var reportedInside int

	h.write(t, func(tx *vault.Txn) error {
		if err := admitter.Admit(tx, posting); err != nil {
			return err
		}
		reportedInside = len(observer.positions)

		return nil
	})

	if reportedInside != 0 {
		t.Fatalf("reported inside the transaction = %d, want none", reportedInside)
	}
	want := yacymodel.DHTRingPositionOfWordPartition(
		yacymodel.WordPartitionOfPosting(posting, h.partitions),
		h.partitions,
	)
	if len(observer.positions) != 1 || observer.positions[0] != want {
		t.Fatalf("reported positions = %v, want [%v]", observer.positions, want)
	}
}

func TestSchedulingAdmitterReportsNothingForAnAlreadyDueWordPartition(t *testing.T) {
	h := openHarness(t)
	observer := &recordingDueObserver{}
	admitter := h.schedule.AdmitterFrom(&fakeAdmitter{}, observer)
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")}
	h.write(t, func(tx *vault.Txn) error { return admitter.Admit(tx, posting) })

	h.write(t, func(tx *vault.Txn) error { return admitter.Admit(tx, posting) })

	if len(observer.positions) != 1 {
		t.Fatalf("reported positions = %v, want one", observer.positions)
	}
}

func TestSchedulingAdmitterReportsNothingWhenTheTransactionFails(t *testing.T) {
	h := openHarness(t)
	observer := &recordingDueObserver{}
	admitter := h.schedule.AdmitterFrom(&fakeAdmitter{}, observer)
	failure := errors.New("commit refused")

	err := h.vault.Update(t.Context(), func(tx *vault.Txn) error {
		if err := admitter.Admit(
			tx,
			yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")},
		); err != nil {
			return err
		}

		return failure
	})

	if !errors.Is(err, failure) {
		t.Fatalf("Update error = %v, want %v", err, failure)
	}
	if len(observer.positions) != 0 {
		t.Fatalf("reported positions = %v, want none", observer.positions)
	}
}
