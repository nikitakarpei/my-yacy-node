package vault_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	firstVersion  = "1"
	secondVersion = "2"
)

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

type dictionary struct {
	vault      *vault.Vault
	words      *vault.Collection[string, string]
	byMeaning  *vault.ValueIndex[definition, definition]
	perMeaning *vault.CountIndex[definition, string]
	changes    *vault.ChangeIndex[definition, definition]
}

func openDictionary(
	t *testing.T,
	engine vault.Engine,
	version string,
	meaningOf func(string) string,
) dictionary {
	t.Helper()

	d := openDictionaryIndexes(t, engine, version, meaningOf)
	words, err := d.vault.RegisterProjectedCollection(
		context.Background(),
		vault.Name("words"),
		stringKeyLayout,
		stringValueCodec{},
		vault.ProjectedAs(definitionOf, d.byMeaning, d.perMeaning, d.changes),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}
	d.words = words

	return d
}

func openDictionaryIndexes(
	t *testing.T,
	engine vault.Engine,
	version string,
	meaningOf func(string) string,
) dictionary {
	t.Helper()

	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	entryOf := func(record definition) definition {
		return definition{word: record.word, meaning: meaningOf(record.meaning)}
	}
	byMeaning, err := v.RegisterValueIndex(
		vault.Name("words_by_meaning"),
		version,
		definitionEntries,
		entryOf,
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	perMeaning, err := v.RegisterCountIndex(
		vault.Name("words_per_meaning"),
		version,
		stringKeyLayout,
		func(record definition) string { return meaningOf(record.meaning) },
	)
	if err != nil {
		t.Fatalf("RegisterCountIndex: %v", err)
	}
	changes, err := v.RegisterChangeIndex(
		vault.Name("words_changed"),
		version,
		definitionEntries,
		entryOf,
	)
	if err != nil {
		t.Fatalf("RegisterChangeIndex: %v", err)
	}

	return dictionary{vault: v, byMeaning: byMeaning, perMeaning: perMeaning, changes: changes}
}

func plainMeaning(meaning string) string { return meaning }

func (d dictionary) define(t *testing.T, definitions ...definition) {
	t.Helper()

	d.write(t, func(tx *vault.Txn) error {
		for _, defined := range definitions {
			if _, err := d.words.Put(tx, defined.word, defined.meaning); err != nil {
				return err
			}
		}

		return nil
	})
}

func (d dictionary) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := d.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (d dictionary) forget(t *testing.T, word string) {
	t.Helper()

	d.write(t, func(tx *vault.Txn) error {
		_, err := d.words.Delete(tx, word)

		return err
	})
}

func (d dictionary) entries(t *testing.T) []string {
	t.Helper()

	var entries []string
	d.read(t, func(tx *vault.Txn) error {
		entries = nil

		return d.byMeaning.Scan(tx, vault.EveryKey(), func(entry definition) (bool, error) {
			entries = append(entries, entry.meaning+"/"+entry.word)

			return true, nil
		})
	})

	return entries
}

func (d dictionary) read(t *testing.T, query func(tx *vault.Txn) error) {
	t.Helper()

	if err := d.vault.View(context.Background(), query); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func (d dictionary) counts(t *testing.T) map[string]int {
	t.Helper()

	counts := map[string]int{}
	d.read(t, func(tx *vault.Txn) error {
		clear(counts)

		return d.perMeaning.Scan(
			tx,
			vault.EveryKey(),
			func(meaning string, count int) (bool, error) {
				counts[meaning] = count

				return true, nil
			},
		)
	})

	return counts
}

func TestAValueIndexHoldsAnEntryPerRecordInEntryOrder(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	d.define(t, definitionOf("b", "x"), definitionOf("a", "y"), definitionOf("c", "x"))

	if got, want := d.entries(t), []string{"x/b", "x/c", "y/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestAValueIndexMovesTheEntryOfAReplacedRecord(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	d.define(t, definitionOf("a", "x"), definitionOf("b", "x"))
	d.define(t, definitionOf("a", "y"), definitionOf("b", "x"))

	if got, want := d.entries(t), []string{"x/b", "y/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestAValueIndexDropsTheEntryOfADeletedRecord(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	d.define(t, definitionOf("a", "x"), definitionOf("b", "y"))
	d.forget(t, "a")
	d.forget(t, "never defined")

	if got, want := d.entries(t), []string{"y/b"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestDeletingAKeyRangeDropsItsRecordsAndTheirIndexEntries(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("a", "x"), definitionOf("b", "y"), definitionOf("c", "x"))

	var amountOfDeleted int
	d.write(t, func(tx *vault.Txn) error {
		var err error
		amountOfDeleted, err = d.words.DeleteWithin(tx, stringKeyParts.KeysFromFirst("b"))

		return err
	})

	if amountOfDeleted != 2 {
		t.Fatalf("deleted records = %d, want 2", amountOfDeleted)
	}
	if got, want := d.entries(t), []string{"x/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	if got, want := d.counts(t), map[string]int{"x": 1}; !maps.Equal(got, want) {
		t.Fatalf("counts = %v, want %v", got, want)
	}
}

func TestDeletingAKeyRangeIsRefusedInsideARead(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("a", "x"))

	err := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		_, err := d.words.DeleteWithin(tx, vault.EveryKey())

		return err
	})

	if err == nil {
		t.Fatal("DeleteWithin inside a read succeeded, want a refusal")
	}
}

func TestACountIndexCountsTheRecordsOfEachGroup(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	d.define(t, definitionOf("a", "x"), definitionOf("b", "x"), definitionOf("c", "y"))
	d.define(t, definitionOf("c", "x"), definitionOf("b", "x"))
	d.forget(t, "a")

	if got, want := d.counts(t), map[string]int{"x": 2}; !maps.Equal(got, want) {
		t.Fatalf("counts = %v, want %v", got, want)
	}
	var emptiedCount int
	d.read(t, func(tx *vault.Txn) error {
		count, err := d.perMeaning.Get(tx, "y")
		emptiedCount = count

		return err
	})
	if emptiedCount != 0 {
		t.Fatalf("count of y = %d, want 0", emptiedCount)
	}
}

func TestRegisteringBuildsIndexesOfRecordsStoredBefore(t *testing.T) {
	engine := newDoubleEngine()
	words := storeWordsWithoutIndexes(t, engine, 2500)

	d := openDictionary(t, engine, firstVersion, plainMeaning)

	if got := len(d.entries(t)); got != len(words) {
		t.Fatalf("entries = %d, want one per stored word (%d)", got, len(words))
	}
	if got, want := d.counts(t), countsOfMeanings(words); !maps.Equal(got, want) {
		t.Fatalf("counts = %v, want %v", got, want)
	}
}

func storeWordsWithoutIndexes(t *testing.T, engine vault.Engine, amount int) []definition {
	t.Helper()

	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	words, err := v.RegisterCollection(vault.Name("words"), stringKeyLayout, stringValueCodec{})
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}
	definitions := numberedDefinitions(amount)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		for _, defined := range definitions {
			if _, err := words.Put(tx, defined.word, defined.meaning); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	return definitions
}

func numberedDefinitions(amount int) []definition {
	definitions := make([]definition, 0, amount)
	for number := range amount {
		definitions = append(definitions, definitionOf(
			fmt.Sprintf("w%05d", number),
			fmt.Sprintf("m%d", number%3),
		))
	}

	return definitions
}

func countsOfMeanings(definitions []definition) map[string]int {
	counts := map[string]int{}
	for _, defined := range definitions {
		counts[defined.meaning]++
	}

	return counts
}

func TestANewVersionRebuildsIndexesFromTheRecords(t *testing.T) {
	engine := newDoubleEngine()
	earlier := openDictionary(t, engine, firstVersion, plainMeaning)
	earlier.define(t, numberedDefinitions(2500)...)

	d := openDictionary(t, engine, secondVersion, strings.ToUpper)

	if got, want := d.counts(
		t,
	), map[string]int{
		"M0": 834,
		"M1": 833,
		"M2": 833,
	}; !maps.Equal(
		got,
		want,
	) {
		t.Fatalf("counts = %v, want %v", got, want)
	}
	entries := d.entries(t)
	if len(entries) != 2500 || !strings.HasPrefix(entries[0], "M0/") {
		t.Fatalf(
			"entries start %v, %d in all, want 2500 entries of upper case meanings",
			entries[:1],
			len(entries),
		)
	}
}

func TestTheSameVersionKeepsIndexesAsTheyAre(t *testing.T) {
	engine := newDoubleEngine()
	earlier := openDictionary(t, engine, firstVersion, plainMeaning)
	earlier.define(t, definitionOf("a", "x"))

	d := openDictionary(t, engine, firstVersion, strings.ToUpper)

	if got, want := d.entries(t), []string{"x/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestAProjectionFollowsOnlyOneCollection(t *testing.T) {
	d := openDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	if _, err := d.vault.RegisterProjectedCollection(
		context.Background(),
		vault.Name("other words"),
		stringKeyLayout,
		stringValueCodec{},
		vault.ProjectedAs(definitionOf, d.byMeaning),
	); err == nil {
		t.Fatal("RegisterProjectedCollection with a followed index succeeded, want error")
	}
}

func TestAnIndexNoCollectionFollowsRefusesReads(t *testing.T) {
	d := openDictionaryIndexes(t, newDoubleEngine(), firstVersion, plainMeaning)

	readErr := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		return d.byMeaning.Scan(tx, vault.EveryKey(), func(definition) (bool, error) {
			return true, nil
		})
	})
	if readErr == nil {
		t.Fatal("Scan of an index no collection follows succeeded, want error")
	}
	countErr := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		_, err := d.perMeaning.Get(tx, "x")

		return err
	})
	if countErr == nil {
		t.Fatal("Get of a count no collection follows succeeded, want error")
	}
}

func TestAProjectedWriteFailsOverARecordThatDoesNotDecode(t *testing.T) {
	engine := newDoubleEngine()
	d := openDictionary(t, engine, firstVersion, plainMeaning)
	engine.plant(vault.Name("words"), stringKeyParts.Key("a").Bytes(), []byte{0xFF})

	putErr := d.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := d.words.Put(tx, "a", "x")

		return err
	})
	if putErr == nil {
		t.Fatal("Put over a record that does not decode succeeded, want error")
	}
	deleteErr := d.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := d.words.Delete(tx, "a")

		return err
	})
	if deleteErr == nil {
		t.Fatal("Delete over a record that does not decode succeeded, want error")
	}
}
