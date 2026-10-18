package vault_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type mention struct {
	word string
	page string
}

var mentionKeyParts = vault.PairKey(vault.TextKeyPart, vault.TextKeyPart)

type mentions struct {
	vault      *vault.Vault
	collection *vault.IndexedCollection[mention, string]
	words      *vault.SecondPartIndex[string, string, mention]
	perPage    *vault.CountIndex[mention, string]
}

func openMentions(t *testing.T, engine vault.Engine) mentions {
	t.Helper()

	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	perPage, err := v.RegisterCountIndex(
		vault.Name("mentions_per_page"),
		firstVersion,
		stringKeyLayout,
		func(stored mention) string { return stored.page },
	)
	if err != nil {
		t.Fatalf("RegisterCountIndex: %v", err)
	}
	words, err := v.RegisterSecondPartIndex(
		vault.Name("mentions_by_page"),
		mentionKeyParts,
		func(stored mention) (string, string) { return stored.word, stored.page },
		func(word, page string) mention { return mention{word: word, page: page} },
	)
	if err != nil {
		t.Fatalf("RegisterSecondPartIndex: %v", err)
	}
	collection, err := v.RegisterIndexedCollection(
		context.Background(),
		vault.Name("mentions"),
		words,
		stringValueCodec{},
		vault.ProjectedAs(func(key mention, _ string) mention { return key }, perPage),
	)
	if err != nil {
		t.Fatalf("RegisterIndexedCollection: %v", err)
	}

	return mentions{vault: v, collection: collection, words: words, perPage: perPage}
}

func mentionsOf(page string, words ...string) []vault.KeyedValue[mention, string] {
	keyedValues := make([]vault.KeyedValue[mention, string], 0, len(words))
	for _, word := range words {
		keyedValues = append(keyedValues, vault.KeyedValue[mention, string]{
			Key:   mention{word: word, page: page},
			Value: word + "@" + page,
		})
	}

	return keyedValues
}

func numberedWords(amount int) []string {
	words := make([]string, 0, amount)
	for number := range amount {
		words = append(words, fmt.Sprintf("w%02d", number))
	}

	return words
}

func (m mentions) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := m.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (m mentions) putEach(t *testing.T, keyedValues []vault.KeyedValue[mention, string]) {
	t.Helper()

	m.write(t, func(tx *vault.Txn) error { return m.collection.PutEach(tx, keyedValues) })
}

func (m mentions) putOneByOne(t *testing.T, keyedValues []vault.KeyedValue[mention, string]) {
	t.Helper()

	m.write(t, func(tx *vault.Txn) error {
		for _, keyedValue := range keyedValues {
			if _, err := m.collection.Put(tx, keyedValue.Key, keyedValue.Value); err != nil {
				return err
			}
		}

		return nil
	})
}

type mentionsState struct {
	length       int
	values       map[mention]string
	wordsPerPage map[string][]string
	countPerPage map[string]int
}

func (m mentions) state(t *testing.T) mentionsState {
	t.Helper()

	state := mentionsState{
		values:       map[mention]string{},
		wordsPerPage: map[string][]string{},
		countPerPage: map[string]int{},
	}
	if err := m.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		if state.length, err = m.collection.Len(tx); err != nil {
			return err
		}
		if err := m.collection.Scan(
			tx,
			vault.EveryKey(),
			func(key mention, value string) (bool, error) {
				state.values[key] = value

				return true, nil
			},
		); err != nil {
			return err
		}
		for _, page := range []string{"p1", "p2", "p3"} {
			if err := m.words.Scan(tx, page, func(key mention) (bool, error) {
				state.wordsPerPage[page] = append(state.wordsPerPage[page], key.word)

				return true, nil
			}); err != nil {
				return err
			}
		}

		return m.perPage.Scan(tx, vault.EveryKey(), func(page string, count int) (bool, error) {
			state.countPerPage[page] = count

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return state
}

func (s mentionsState) equals(other mentionsState) bool {
	return s.length == other.length &&
		maps.Equal(s.values, other.values) &&
		maps.EqualFunc(s.wordsPerPage, other.wordsPerPage, slices.Equal) &&
		maps.Equal(s.countPerPage, other.countPerPage)
}

func assertPutEachMatchesPut(
	t *testing.T,
	stored, written []vault.KeyedValue[mention, string],
) {
	t.Helper()

	byPut, byPutEach := openMentions(t, newDoubleEngine()), openMentions(t, newDoubleEngine())
	byPut.putOneByOne(t, stored)
	byPutEach.putOneByOne(t, stored)
	byPut.putOneByOne(t, written)
	byPutEach.putEach(t, written)

	if want, got := byPut.state(t), byPutEach.state(t); !got.equals(want) {
		t.Fatalf("after PutEach %+v, after Put %+v", got, want)
	}
}

func TestPutEachStoresTheWordsOfANewPageAsPutWould(t *testing.T) {
	assertPutEachMatchesPut(t, nil, mentionsOf("p1", numberedWords(20)...))
}

func TestPutEachReplacesTheWordsAPageAlreadyHeldAsPutWould(t *testing.T) {
	assertPutEachMatchesPut(
		t,
		mentionsOf("p1", numberedWords(5)...),
		append(mentionsOf("p1", numberedWords(20)...), mentionsOf("p2", "w01")...),
	)
}

func TestPutEachStoresARepeatedWordOnceAsPutWould(t *testing.T) {
	written := mentionsOf("p1", numberedWords(12)...)
	written = append(written, vault.KeyedValue[mention, string]{
		Key:   mention{word: "w03", page: "p1"},
		Value: "again",
	})

	assertPutEachMatchesPut(t, nil, written)
}

func TestPutEachStoresPagesOfFewWordsAsPutWould(t *testing.T) {
	assertPutEachMatchesPut(
		t,
		mentionsOf("p2", "w01"),
		append(mentionsOf("p2", "w01", "w02"), mentionsOf("p3", "w01")...),
	)
}

func TestPutEachStoresAgainTheWordsDeletedEarlierInTheTransaction(t *testing.T) {
	m := openMentions(t, newDoubleEngine())
	m.putOneByOne(t, mentionsOf("p1", numberedWords(10)...))

	m.write(t, func(tx *vault.Txn) error {
		for _, keyedValue := range mentionsOf("p1", numberedWords(10)...) {
			if _, err := m.collection.Delete(tx, keyedValue.Key); err != nil {
				return err
			}
		}

		return m.collection.PutEach(tx, mentionsOf("p1", numberedWords(10)...))
	})

	if got := m.state(t).countPerPage["p1"]; got != 10 {
		t.Fatalf("count of p1 = %d, want 10", got)
	}
}

func TestPutEachOutsideAWriteIsRefused(t *testing.T) {
	m := openMentions(t, newDoubleEngine())

	err := m.vault.View(context.Background(), func(tx *vault.Txn) error {
		return m.collection.PutEach(tx, mentionsOf("p1", numberedWords(10)...))
	})
	if err == nil {
		t.Fatal("PutEach in a view succeeded, want refusal")
	}
}

func TestRegisteringBuildsTheSecondPartIndexOfRecordsStoredBefore(t *testing.T) {
	engine := newDoubleEngine()
	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain, err := v.RegisterCollection(
		vault.Name("mentions"),
		mentionKeyParts.KeyLayoutFor(
			func(stored mention) (string, string) { return stored.word, stored.page },
			func(word, page string) mention { return mention{word: word, page: page} },
		),
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		for _, keyedValue := range mentionsOf("p1", numberedWords(10)...) {
			if _, err := plain.Put(tx, keyedValue.Key, keyedValue.Value); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	m := openMentions(t, engine)

	if got, want := m.state(t).wordsPerPage["p1"], numberedWords(10); !slices.Equal(got, want) {
		t.Fatalf("words of p1 = %v, want %v", got, want)
	}
}

func TestAnIndexedCollectionWithoutProjectionsKeepsItsSecondPartIndex(t *testing.T) {
	v, err := vault.New(newDoubleEngine(), nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	words, err := v.RegisterSecondPartIndex(
		vault.Name("mentions_by_page"),
		mentionKeyParts,
		func(stored mention) (string, string) { return stored.word, stored.page },
		func(word, page string) mention { return mention{word: word, page: page} },
	)
	if err != nil {
		t.Fatalf("RegisterSecondPartIndex: %v", err)
	}
	collection, err := v.RegisterIndexedCollection(
		context.Background(),
		vault.Name("mentions"),
		words,
		stringValueCodec{},
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterIndexedCollection: %v", err)
	}
	var wordsOfPage []string
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		if err := collection.PutEach(tx, mentionsOf("p1", "a", "b")); err != nil {
			return err
		}
		if _, err := collection.Delete(tx, mention{word: "a", page: "p1"}); err != nil {
			return err
		}

		return words.Scan(tx, "p1", func(key mention) (bool, error) {
			wordsOfPage = append(wordsOfPage, key.word)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !slices.Equal(wordsOfPage, []string{"b"}) {
		t.Fatalf("words of p1 = %v, want [b]", wordsOfPage)
	}
}
