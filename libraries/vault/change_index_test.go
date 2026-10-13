package vault_test

import (
	"context"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func initialOf(meaning string) string { return meaning[:1] }

func (d dictionary) pendingChanges(t *testing.T) []vault.Change[definition] {
	t.Helper()

	var changes []vault.Change[definition]
	d.read(t, func(tx *vault.Txn) error {
		changes = nil

		return d.changes.Scan(
			tx,
			vault.EveryKey(),
			func(change vault.Change[definition]) (bool, error) {
				changes = append(changes, change)

				return true, nil
			},
		)
	})

	return changes
}

func (d dictionary) changedEntries(t *testing.T) []string {
	t.Helper()

	changes := d.pendingChanges(t)
	entries := make([]string, 0, len(changes))
	for _, change := range changes {
		entries = append(entries, change.Entry.meaning+"/"+change.Entry.word)
	}

	return entries
}

func (d dictionary) acknowledge(t *testing.T, change vault.Change[definition]) bool {
	t.Helper()

	var wasAcknowledged bool
	d.write(t, func(tx *vault.Txn) error {
		acknowledged, err := d.changes.Acknowledge(tx, change)
		wasAcknowledged = acknowledged

		return err
	})

	return wasAcknowledged
}

func (d dictionary) acknowledgeEach(t *testing.T, changes []vault.Change[definition]) {
	t.Helper()

	for _, change := range changes {
		if !d.acknowledge(t, change) {
			t.Fatalf("Acknowledge(%v) = false, want true", change.Entry)
		}
	}
}

func (d dictionary) amountOfChanges(t *testing.T) int {
	t.Helper()

	var amount int
	d.read(t, func(tx *vault.Txn) error {
		length, err := d.changes.Len(tx)
		amount = length

		return err
	})

	return amount
}

func TestAChangeIndexHoldsAnEntryPerAddedRecord(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)

	d.define(t, definitionOf("b", "y1"), definitionOf("a", "x1"))

	if got, want := d.changedEntries(t), []string{"x/a", "y/b"}; !slices.Equal(got, want) {
		t.Fatalf("changed entries = %v, want %v", got, want)
	}
	if got := d.amountOfChanges(t); got != 2 {
		t.Fatalf("Len = %d, want 2", got)
	}
}

func TestAnAcknowledgedChangeLeavesTheIndex(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"), definitionOf("b", "y1"))

	d.acknowledgeEach(t, d.pendingChanges(t))

	if got := d.changedEntries(t); len(got) != 0 {
		t.Fatalf("changed entries = %v, want none", got)
	}
	if got := d.amountOfChanges(t); got != 0 {
		t.Fatalf("Len = %d, want 0", got)
	}
}

func TestARecordReplacedAfterItsScanStaysChanged(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"))
	scanned := d.pendingChanges(t)

	d.define(t, definitionOf("a", "x2"))

	if d.acknowledge(t, scanned[0]) {
		t.Fatal("Acknowledge of a change scanned before the replace = true, want false")
	}
	if got, want := d.changedEntries(t), []string{"x/a"}; !slices.Equal(got, want) {
		t.Fatalf("changed entries = %v, want %v", got, want)
	}
	d.acknowledgeEach(t, d.pendingChanges(t))
	if got := d.amountOfChanges(t); got != 0 {
		t.Fatalf("Len = %d, want 0", got)
	}
}

func TestReplacingAnAcknowledgedRecordChangesItAgain(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"), definitionOf("b", "y1"))
	d.acknowledgeEach(t, d.pendingChanges(t))

	d.define(t, definitionOf("a", "x2"), definitionOf("b", "z1"))

	if got, want := d.changedEntries(t), []string{"x/a", "z/b"}; !slices.Equal(got, want) {
		t.Fatalf("changed entries = %v, want %v", got, want)
	}
}

func TestReplacingARecordUnderAnotherEntryMovesItsChange(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"))
	scanned := d.pendingChanges(t)

	d.define(t, definitionOf("a", "y1"))

	if got, want := d.changedEntries(t), []string{"y/a"}; !slices.Equal(got, want) {
		t.Fatalf("changed entries = %v, want %v", got, want)
	}
	if d.acknowledge(t, scanned[0]) {
		t.Fatal("Acknowledge of the moved entry = true, want false")
	}
	if got := d.amountOfChanges(t); got != 1 {
		t.Fatalf("Len = %d, want 1", got)
	}
}

func TestRemovingARecordRemovesItsChange(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"), definitionOf("b", "y1"))
	d.acknowledgeEach(t, d.pendingChanges(t)[:1])

	d.forget(t, "a")
	d.forget(t, "b")

	if got := d.amountOfChanges(t); got != 0 {
		t.Fatalf("Len = %d, want 0", got)
	}
	recordCounts, err := d.vault.RecordCountsByBucket(context.Background())
	if err != nil {
		t.Fatalf("RecordCountsByBucket: %v", err)
	}
	if got := recordCounts[vault.Name("words_changed")]; got != 0 {
		t.Fatalf("records of words_changed = %d, want 0", got)
	}
}

func TestAChangeAcknowledgesTheSameRecordRemovedAndAddedAgain(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"))
	scanned := d.pendingChanges(t)

	d.forget(t, "a")
	d.define(t, definitionOf("a", "x1"))

	if !d.acknowledge(t, scanned[0]) {
		t.Fatal("Acknowledge of the same record added again = false, want true")
	}
	if got := d.amountOfChanges(t); got != 0 {
		t.Fatalf("Len = %d, want 0", got)
	}
}

func TestANewVersionChangesEveryRecord(t *testing.T) {
	engine := newDoubleEngine()
	earlier := openDictionary(t, engine, firstVersion, initialOf)
	earlier.define(t, definitionOf("a", "x1"), definitionOf("b", "y1"))
	earlier.acknowledgeEach(t, earlier.pendingChanges(t))

	same := openDictionary(t, engine, firstVersion, initialOf)
	if got := same.changedEntries(t); len(got) != 0 {
		t.Fatalf("changed entries under the same version = %v, want none", got)
	}
	d := openDictionary(t, engine, secondVersion, initialOf)

	if got, want := d.changedEntries(t), []string{"x/a", "y/b"}; !slices.Equal(got, want) {
		t.Fatalf("changed entries = %v, want %v", got, want)
	}
}

func TestAChangeIndexNoCollectionFollowsRefusesUse(t *testing.T) {
	d := openDictionaryIndexes(t, newDoubleEngine(), firstVersion, initialOf)

	scanErr := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		return d.changes.Scan(tx, vault.EveryKey(), func(vault.Change[definition]) (bool, error) {
			return true, nil
		})
	})
	if scanErr == nil {
		t.Fatal("Scan of a change index no collection follows succeeded, want error")
	}
	acknowledgeErr := d.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := d.changes.Acknowledge(tx, vault.Change[definition]{})

		return err
	})
	if acknowledgeErr == nil {
		t.Fatal("Acknowledge on a change index no collection follows succeeded, want error")
	}
	lenErr := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		_, err := d.changes.Len(tx)

		return err
	})
	if lenErr == nil {
		t.Fatal("Len of a change index no collection follows succeeded, want error")
	}
}

func TestAcknowledgingInsideAReadOnlyTransactionFails(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)
	d.define(t, definitionOf("a", "x1"))
	scanned := d.pendingChanges(t)

	readErr := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		_, err := d.changes.Acknowledge(tx, scanned[0])

		return err
	})
	if readErr == nil {
		t.Fatal("Acknowledge inside View succeeded, want error")
	}
}

func TestAcknowledgingAnEntryNeverChangedFindsNothing(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, initialOf)

	if d.acknowledge(t, vault.Change[definition]{Entry: definitionOf("a", "x")}) {
		t.Fatal("Acknowledge of an absent entry = true, want false")
	}
}
