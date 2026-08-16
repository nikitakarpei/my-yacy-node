package vault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type recordingObserver struct {
	begins        int
	beginRefusals []vault.WriteRefusalCause
	committed     int
	writing       int
	aborted       int
	refused       int
	refusalCauses []vault.WriteRefusalCause
	executeCount  int
	closeCount    int
	readsBegan    int
	readsEnded    int
}

func newRecordingObserver() *recordingObserver {
	return &recordingObserver{}
}

func (o *recordingObserver) ObserveWriteBegan(time.Duration) { o.begins++ }

func (o *recordingObserver) ObserveWriteBeginRefused(cause vault.WriteRefusalCause) {
	o.beginRefusals = append(o.beginRefusals, cause)
}

func (o *recordingObserver) ObserveWriteCommitted(_, _ time.Duration, calledWriteOperation bool) {
	o.committed++
	if calledWriteOperation {
		o.writing++
	}
	o.executeCount++
	o.closeCount++
}

func (o *recordingObserver) ObserveWriteAborted(_, _ time.Duration) {
	o.aborted++
	o.executeCount++
	o.closeCount++
}

func (o *recordingObserver) ObserveWriteCommitRefused(
	_, _ time.Duration,
	cause vault.WriteRefusalCause,
) {
	o.refused++
	o.executeCount++
	o.closeCount++
	o.refusalCauses = append(o.refusalCauses, cause)
}

func (o *recordingObserver) ObserveReadBegan() { o.readsBegan++ }

func (o *recordingObserver) ObserveReadEnded() { o.readsEnded++ }

type refusingEngine struct {
	doubleEngine

	failure error
}

func (e *refusingEngine) Update(context.Context, func(vault.EngineTxn) error) error {
	return e.failure
}

type commitRefusingEngine struct {
	doubleEngine

	failure error
}

func (e *commitRefusingEngine) Update(_ context.Context, fn func(vault.EngineTxn) error) error {
	if err := fn(doubleTxn{buckets: e.buckets, writable: true}); err != nil {
		return err
	}

	return e.failure
}

type noSpaceError struct{}

func (noSpaceError) Error() string { return "at capacity" }

func (noSpaceError) Cause() vault.WriteRefusalCause { return "no_space" }

func openObserved(t *testing.T, engine vault.Engine) (*vault.Vault, *recordingObserver) {
	t.Helper()

	observer := newRecordingObserver()

	v, err := vault.New(engine, observer)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return v, observer
}

func TestCommittedWriteReportsBeginAndCommit(t *testing.T) {
	v, observer := openObserved(t, newDoubleEngine())

	if err := v.Update(context.Background(), func(*vault.Txn) error { return nil }); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if observer.begins != 1 {
		t.Errorf("begins = %d, want 1", observer.begins)
	}
	if observer.committed != 1 {
		t.Errorf("committed = %d, want 1", observer.committed)
	}
	if observer.aborted != 0 || observer.refused != 0 {
		t.Errorf("aborted = %d, refused = %d, want 0 and 0", observer.aborted, observer.refused)
	}
}

func TestCommittedWriteThatStoresReportsWriteOperation(t *testing.T) {
	v, observer, words := openObservedWords(t)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if observer.committed != 1 || observer.writing != 1 {
		t.Errorf(
			"committed = %d, writing = %d, want 1 and 1",
			observer.committed,
			observer.writing,
		)
	}
}

func TestCommittedWriteThatOnlyReadsReportsNoWriteOperation(t *testing.T) {
	v, observer, words := openObservedWords(t)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		_, _, err := words.Get(tx, "a")

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if observer.committed != 1 || observer.writing != 0 {
		t.Errorf(
			"committed = %d, writing = %d, want 1 and 0",
			observer.committed,
			observer.writing,
		)
	}
}

func TestCommittedWriteDeletingAbsentKeyReportsWriteOperation(t *testing.T) {
	v, observer, words := openObservedWords(t)

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := words.Delete(tx, "a")

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if observer.committed != 1 || observer.writing != 1 {
		t.Errorf(
			"committed = %d, writing = %d, want 1 and 1",
			observer.committed,
			observer.writing,
		)
	}
}

func openObservedWords(
	t *testing.T,
) (*vault.Vault, *recordingObserver, *vault.Collection[string, string]) {
	t.Helper()

	v, observer := openObserved(t, newDoubleEngine())

	words, err := vault.RegisterCollection(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	return v, observer, words
}

func TestAbortedClosureReportsAborted(t *testing.T) {
	v, observer := openObserved(t, newDoubleEngine())

	aborted := errors.New("caller aborted")
	if err := v.Update(context.Background(), func(*vault.Txn) error {
		return aborted
	}); !errors.Is(err, aborted) {
		t.Fatalf("Update error = %v, want %v", err, aborted)
	}

	if observer.aborted != 1 {
		t.Errorf("aborted = %d, want 1", observer.aborted)
	}
	if observer.committed != 0 || observer.refused != 0 {
		t.Errorf(
			"committed = %d, refused = %d, want 0 and 0 for an aborted closure",
			observer.committed,
			observer.refused,
		)
	}
	if len(observer.refusalCauses) != 0 {
		t.Errorf("refusal causes = %v, want none", observer.refusalCauses)
	}
}

func TestBeginRefusalReportsCauseWithoutOpeningTransaction(t *testing.T) {
	engine := &refusingEngine{doubleEngine: *newDoubleEngine(), failure: noSpaceError{}}
	v, observer := openObserved(t, engine)

	if err := v.Update(context.Background(), func(*vault.Txn) error {
		return nil
	}); err == nil {
		t.Fatal("Update succeeded, want failure")
	}

	if len(observer.beginRefusals) != 1 || observer.beginRefusals[0] != "no_space" {
		t.Errorf("begin refusals = %v, want [no_space]", observer.beginRefusals)
	}
	if observer.begins != 0 {
		t.Errorf("begins = %d, want 0: the closure never ran", observer.begins)
	}
	if observer.committed != 0 || observer.aborted != 0 || observer.refused != 0 {
		t.Errorf("a begin refusal is not a terminal outcome")
	}
}

func TestUncarriedBeginRefusalReportsUnclassifiedCause(t *testing.T) {
	engine := &refusingEngine{
		doubleEngine: *newDoubleEngine(),
		failure:      errors.New("engine boom"),
	}
	v, observer := openObserved(t, engine)

	if err := v.Update(context.Background(), func(*vault.Txn) error {
		return nil
	}); err == nil {
		t.Fatal("Update succeeded, want failure")
	}

	if len(observer.beginRefusals) != 1 || observer.beginRefusals[0] != "unclassified" {
		t.Errorf("begin refusals = %v, want [unclassified]", observer.beginRefusals)
	}
}

func TestCommitRefusalAfterClosureSucceedsReportsRefused(t *testing.T) {
	engine := &commitRefusingEngine{doubleEngine: *newDoubleEngine(), failure: noSpaceError{}}
	v, observer := openObserved(t, engine)

	if err := v.Update(context.Background(), func(*vault.Txn) error {
		return nil
	}); err == nil {
		t.Fatal("Update succeeded, want failure")
	}

	if observer.refused != 1 {
		t.Errorf(
			"refused = %d, want 1: the closure ran and the engine still failed to commit",
			observer.refused,
		)
	}
	if observer.committed != 0 {
		t.Errorf("committed = %d, want 0", observer.committed)
	}
	if len(observer.refusalCauses) != 1 || observer.refusalCauses[0] != "no_space" {
		t.Errorf("refusal causes = %v, want [no_space]", observer.refusalCauses)
	}
	if observer.begins != 1 {
		t.Errorf("begins = %d, want 1: the transaction did open", observer.begins)
	}
}

type skippingEngine struct {
	doubleEngine
}

func (e *skippingEngine) Update(context.Context, func(vault.EngineTxn) error) error {
	return nil
}

func TestEngineSkippingClosureCannotReportSuccess(t *testing.T) {
	engine := &skippingEngine{doubleEngine: *newDoubleEngine()}
	v, observer := openObserved(t, engine)

	if err := v.Update(context.Background(), func(*vault.Txn) error {
		return nil
	}); err == nil {
		t.Fatal("Update succeeded, want failure: the engine never opened a transaction")
	}

	if observer.committed != 0 {
		t.Errorf("committed = %d, want 0", observer.committed)
	}
	if len(observer.beginRefusals) != 1 {
		t.Errorf("begin refusals = %d, want 1", len(observer.beginRefusals))
	}
}

func TestReadsInFlightBalanceAcrossView(t *testing.T) {
	v, observer := openObserved(t, newDoubleEngine())

	if err := v.View(context.Background(), func(*vault.Txn) error {
		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	if observer.readsBegan != 1 || observer.readsEnded != 1 {
		t.Errorf(
			"reads began = %d, ended = %d, want 1 and 1",
			observer.readsBegan,
			observer.readsEnded,
		)
	}
}

func TestUsedBytesIsNotAReadInFlight(t *testing.T) {
	v, observer := openObserved(t, newDoubleEngine())

	if _, err := v.UsedBytes(context.Background()); err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	if observer.readsBegan != 0 {
		t.Errorf(
			"reads began = %d, want 0: a capacity reading is not a read",
			observer.readsBegan,
		)
	}
}

func TestEntriesByCollectionIsNotAReadInFlight(t *testing.T) {
	v, observer := openObserved(t, newDoubleEngine())

	if _, err := v.EntriesByCollection(context.Background()); err != nil {
		t.Fatalf("EntriesByCollection: %v", err)
	}

	if observer.readsBegan != 0 {
		t.Errorf(
			"reads began = %d, want 0: a metrics scrape is not a read",
			observer.readsBegan,
		)
	}
}
