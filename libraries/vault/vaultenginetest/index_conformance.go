package vaultenginetest

import (
	"context"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func runIndexConformance(t *testing.T, open func(quotaBytes int64) (vault.Engine, error)) {
	t.Helper()

	t.Run(
		"IndexEntriesFollowTheirRecords",
		func(t *testing.T) { indexEntriesFollowTheirRecords(t, open) },
	)
	t.Run(
		"IndexEntriesFollowRecordsChangedTwiceInOneTransaction",
		func(t *testing.T) { indexEntriesFollowRecordsChangedTwiceInOneTransaction(t, open) },
	)
	t.Run(
		"ChangeEntriesKeepRecordCounts",
		func(t *testing.T) { changeEntriesKeepRecordCounts(t, open) },
	)
}

type definition struct {
	word    string
	meaning string
}

func definitionOf(word, meaning string) definition {
	return definition{word: word, meaning: meaning}
}

var definitionEntries = vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).KeyLayoutFor(
	func(entry definition) (string, string) { return entry.meaning, entry.word },
	func(meaning, word string) definition { return definition{word: word, meaning: meaning} },
)

type glossary struct {
	vault     *vault.Vault
	words     *vault.Collection[string, string]
	byMeaning *vault.ValueIndex[definition, definition]
	changes   *vault.ChangeIndex[definition, definition]
}

func openGlossary(t *testing.T, open func(int64) (vault.Engine, error)) glossary {
	t.Helper()

	v := openVault(t, open, 0)
	byMeaning, err := v.RegisterValueIndex(
		vault.Name("words_by_meaning"),
		"1",
		definitionEntries,
		func(record definition) definition { return record },
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	changes, err := v.RegisterChangeIndex(
		vault.Name("words_changed"),
		"1",
		definitionEntries,
		func(record definition) definition { return record },
	)
	if err != nil {
		t.Fatalf("RegisterChangeIndex: %v", err)
	}
	words, err := v.RegisterProjectedCollection(
		context.Background(),
		vault.Name("words"),
		stringKeyLayout,
		stringValueCodec{},
		vault.ProjectedAs(definitionOf, byMeaning, changes),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}

	return glossary{vault: v, words: words, byMeaning: byMeaning, changes: changes}
}

func (g glossary) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := g.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (g glossary) definitionsByMeaning(t *testing.T) []definition {
	t.Helper()

	var definitions []definition
	if err := g.vault.View(context.Background(), func(tx *vault.Txn) error {
		return g.byMeaning.Scan(tx, vault.EveryKey(), func(entry definition) (bool, error) {
			definitions = append(definitions, entry)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return definitions
}

func indexEntriesFollowTheirRecords(t *testing.T, open func(int64) (vault.Engine, error)) {
	g := openGlossary(t, open)
	g.write(t, func(tx *vault.Txn) error {
		for _, defined := range []definition{
			definitionOf("a", "x"),
			definitionOf("b", "y"),
			definitionOf("c", "x"),
		} {
			if _, err := g.words.Put(tx, defined.word, defined.meaning); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	})
	g.write(t, func(tx *vault.Txn) error {
		if _, err := g.words.Put(tx, "b", "x"); err != nil {
			return wrapTest(err)
		}
		if _, err := g.words.Put(tx, "c", "x"); err != nil {
			return wrapTest(err)
		}
		if _, err := g.words.Delete(tx, "a"); err != nil {
			return wrapTest(err)
		}

		return nil
	})

	got := g.definitionsByMeaning(t)
	want := []definition{definitionOf("b", "x"), definitionOf("c", "x")}
	if !slices.Equal(got, want) {
		t.Fatalf("definitions by meaning = %v, want %v", got, want)
	}
}

func indexEntriesFollowRecordsChangedTwiceInOneTransaction(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	g := openGlossary(t, open)
	g.write(t, func(tx *vault.Txn) error {
		if _, err := g.words.Put(tx, "d", "z"); err != nil {
			return wrapTest(err)
		}
		if _, err := g.words.Put(tx, "d", "w"); err != nil {
			return wrapTest(err)
		}
		if _, err := g.words.Put(tx, "e", "v"); err != nil {
			return wrapTest(err)
		}
		if _, err := g.words.Delete(tx, "e"); err != nil {
			return wrapTest(err)
		}

		return nil
	})

	got := g.definitionsByMeaning(t)
	want := []definition{definitionOf("d", "w")}
	if !slices.Equal(got, want) {
		t.Fatalf("definitions by meaning = %v, want %v", got, want)
	}
}

func changeEntriesKeepRecordCounts(t *testing.T, open func(int64) (vault.Engine, error)) {
	g := openGlossary(t, open)
	g.define(
		t,
		definitionOf("a", "x"),
		definitionOf("b", "y"),
		definitionOf("c", "x"),
		definitionOf("d", "w"),
	)
	g.acknowledge(t, g.pendingChanges(t)[1])
	g.define(t, definitionOf("b", "z"), definitionOf("c", "x"))
	g.write(t, func(tx *vault.Txn) error {
		for _, word := range []string{"a", "b", "c"} {
			if _, err := g.words.Delete(tx, word); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	})

	recordCounts, err := g.vault.RecordCountsByBucket(context.Background())
	if err != nil {
		t.Fatalf("RecordCountsByBucket: %v", err)
	}
	if got, want := recordCounts[vault.Name("words_changed")], len(
		g.pendingChanges(t),
	); got != want {
		t.Fatalf("records of words_changed = %d, want one per pending change (%d)", got, want)
	}
}

func (g glossary) define(t *testing.T, definitions ...definition) {
	t.Helper()

	g.write(t, func(tx *vault.Txn) error {
		for _, defined := range definitions {
			if _, err := g.words.Put(tx, defined.word, defined.meaning); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	})
}

func (g glossary) pendingChanges(t *testing.T) []definition {
	t.Helper()

	var changes []definition
	if err := g.vault.View(context.Background(), func(tx *vault.Txn) error {
		changes = nil

		return g.changes.Scan(
			tx,
			vault.EveryKey(),
			func(change definition) (bool, error) {
				changes = append(changes, change)

				return true, nil
			},
		)
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return changes
}

func (g glossary) acknowledge(t *testing.T, change definition) {
	t.Helper()

	g.write(t, func(tx *vault.Txn) error {
		if _, err := g.changes.Acknowledge(tx, change); err != nil {
			return wrapTest(err)
		}

		return nil
	})
}
