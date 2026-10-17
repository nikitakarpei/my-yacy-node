package wordpartitionschedule_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type fakeWriter struct {
	admitted []yacymodel.RWIPosting
	err      error
}

func (f *fakeWriter) Write(_ *vault.Txn, postings ...yacymodel.RWIPosting) error {
	if f.err != nil {
		return f.err
	}
	f.admitted = append(f.admitted, postings...)

	return nil
}

func TestSchedulingAdmitterStoresThePostingAndCountsItsAdmission(t *testing.T) {
	h := openHarness(t)
	wrapped := &fakeWriter{}
	writer := h.schedule.WriterFrom(wrapped, &recordingAdmissionObserver{})
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")}

	h.write(t, func(tx *vault.Txn) error { return writer.Write(tx, posting) })
	h.scheduleAdmissionsWithin(t, wholeRing)

	if len(wrapped.admitted) != 1 || wrapped.admitted[0] != posting {
		t.Fatalf("wrapped writer got %v, want [%v]", wrapped.admitted, posting)
	}
	want := yacymodel.WordPartitionOfPosting(posting, h.partitions)
	if due := h.dueWordPartitions(t, 10); !slices.Equal(due, []yacymodel.WordPartition{want}) {
		t.Fatalf("due word partitions = %v, want [%v]", due, want)
	}
}

func TestSchedulingAdmitterCountsEachPostingAdmittedAtOnce(t *testing.T) {
	h := openHarness(t)
	observer := &recordingAdmissionObserver{}
	writer := h.schedule.WriterFrom(&fakeWriter{}, observer)
	postings := []yacymodel.RWIPosting{
		{WordHash: yacymodel.WordHash("first")},
		{WordHash: yacymodel.WordHash("second")},
	}

	h.write(t, func(tx *vault.Txn) error { return writer.Write(tx, postings...) })
	h.scheduleAdmissionsWithin(t, wholeRing)

	if len(observer.positions) != len(postings) {
		t.Fatalf("reported positions = %v, want one per posting", observer.positions)
	}
	if due := h.dueWordPartitions(t, 10); len(due) != len(postings) {
		t.Fatalf("due word partitions = %v, want one per posting", due)
	}
}

func TestSchedulingAdmitterReturnsWrappedErrorAndCountsNothing(t *testing.T) {
	h := openHarness(t)
	failure := errors.New("admit failed")
	writer := h.schedule.WriterFrom(
		&fakeWriter{err: failure},
		&recordingAdmissionObserver{},
	)

	err := h.vault.Update(t.Context(), func(tx *vault.Txn) error {
		return writer.Write(tx, yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")})
	})
	h.scheduleAdmissionsWithin(t, wholeRing)

	if !errors.Is(err, failure) {
		t.Fatalf("Admit error = %v, want %v", err, failure)
	}
	if due := h.dueWordPartitions(t, 10); len(due) != 0 {
		t.Fatalf("due word partitions = %v, want none", due)
	}
}

type recordingAdmissionObserver struct {
	positions []yacymodel.DHTRingPosition
}

func (r *recordingAdmissionObserver) ObservePostingAdmitted(position yacymodel.DHTRingPosition) {
	r.positions = append(r.positions, position)
}

func TestSchedulingAdmitterReportsThePositionOnlyAfterCommit(t *testing.T) {
	h := openHarness(t)
	observer := &recordingAdmissionObserver{}
	writer := h.schedule.WriterFrom(&fakeWriter{}, observer)
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")}
	var reportedInside int

	h.write(t, func(tx *vault.Txn) error {
		if err := writer.Write(tx, posting); err != nil {
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

func TestSchedulingAdmitterReportsEveryAdmittedPosting(t *testing.T) {
	h := openHarness(t)
	observer := &recordingAdmissionObserver{}
	writer := h.schedule.WriterFrom(&fakeWriter{}, observer)
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("word")}
	h.write(t, func(tx *vault.Txn) error { return writer.Write(tx, posting) })

	h.write(t, func(tx *vault.Txn) error { return writer.Write(tx, posting) })

	if len(observer.positions) != 2 {
		t.Fatalf("reported positions = %v, want two", observer.positions)
	}
}

func TestSchedulingAdmitterReportsNothingWhenTheTransactionFails(t *testing.T) {
	h := openHarness(t)
	observer := &recordingAdmissionObserver{}
	writer := h.schedule.WriterFrom(&fakeWriter{}, observer)
	failure := errors.New("commit refused")

	err := h.vault.Update(t.Context(), func(tx *vault.Txn) error {
		if err := writer.Write(
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
