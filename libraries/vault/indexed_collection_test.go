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

var rankedMentionWords = []string{"w00", "w03"}

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

func (p pageWords) putEach(t *testing.T, keyedValues []vault.KeyedValue[mention, string]) {
	t.Helper()

	p.write(t, func(tx *vault.Txn) error { return p.collection.PutEach(tx, keyedValues) })
}

func (p pageWords) putOneByOne(t *testing.T, keyedValues []vault.KeyedValue[mention, string]) {
	t.Helper()

	p.write(t, func(tx *vault.Txn) error {
		for _, keyedValue := range keyedValues {
			if _, err := p.collection.Put(tx, keyedValue.Key, keyedValue.Value); err != nil {
				return err
			}
		}

		return nil
	})
}

type pageWordsState struct {
	length       int
	values       map[mention]string
	wordsPerPage map[string][]string
	countPerPage map[string]int
}

func (p pageWords) state(t *testing.T) pageWordsState {
	t.Helper()

	state := pageWordsState{
		values:       map[mention]string{},
		wordsPerPage: map[string][]string{},
		countPerPage: map[string]int{},
	}
	if err := p.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		if state.length, err = p.collection.Len(tx); err != nil {
			return err
		}
		if err := p.collection.Scan(
			tx,
			vault.EveryKey(),
			func(key mention, value string) (bool, error) {
				state.values[key] = value

				return true, nil
			},
		); err != nil {
			return err
		}

		return p.perPage.Scan(tx, vault.EveryKey(), func(page string, count int) (bool, error) {
			state.countPerPage[page] = count

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	for _, page := range []string{"p1", "p2", "p3"} {
		if words := p.wordsOf(t, page); len(words) > 0 {
			state.wordsPerPage[page] = words
		}
	}

	return state
}

func (s pageWordsState) equals(other pageWordsState) bool {
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

	byPut := openPageWords(t, newDoubleEngine(), rankedMentionWords)
	byPutEach := openPageWords(t, newDoubleEngine(), rankedMentionWords)
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
	p := openPageWords(t, newDoubleEngine(), rankedMentionWords)
	p.putOneByOne(t, mentionsOf("p1", numberedWords(10)...))

	p.write(t, func(tx *vault.Txn) error {
		if err := p.forget(tx, "p1", numberedWords(10)...); err != nil {
			return err
		}

		return p.collection.PutEach(tx, mentionsOf("p1", numberedWords(10)...))
	})

	if got := p.state(t).countPerPage["p1"]; got != 10 {
		t.Fatalf("count of p1 = %d, want 10", got)
	}
}

func TestPutEachOutsideAWriteIsRefused(t *testing.T) {
	p := openPageWords(t, newDoubleEngine(), rankedMentionWords)

	err := p.vault.View(context.Background(), func(tx *vault.Txn) error {
		return p.collection.PutEach(tx, mentionsOf("p1", numberedWords(10)...))
	})
	if err == nil {
		t.Fatal("PutEach in a view succeeded, want refusal")
	}
}

func TestRegisteringBuildsTheIndexOfRecordsStoredBefore(t *testing.T) {
	engine := newDoubleEngine()
	v, err := vault.New(engine, nil, nil, newFakeClock(testEpoch))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain, err := v.RegisterCollection(
		pageMentionsBucket,
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

	p := openPageWords(t, engine, rankedMentionWords)

	assertWords(t, p.wordsOf(t, "p1"), numberedWords(10)...)
}

func TestAnIndexedCollectionWithoutProjectionsKeepsItsIndex(t *testing.T) {
	v, err := vault.New(newDoubleEngine(), nil, nil, newFakeClock(testEpoch))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dictionary, err := vault.FixedDictionaryFrom(vault.TextKeyPart, rankedMentionWords)
	if err != nil {
		t.Fatalf("FixedDictionaryFrom: %v", err)
	}
	words, err := v.RegisterMemberSetIndex(
		pageWordsBucket,
		firstVersion,
		mentionKeyParts,
		func(stored mention) (string, string) { return stored.word, stored.page },
		func(word, page string) mention { return mention{word: word, page: page} },
		vault.MemberEncodingFrom(dictionary),
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterMemberSetIndex: %v", err)
	}
	collection, err := v.RegisterIndexedCollection(
		context.Background(),
		pageMentionsBucket,
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
		wordsOfPage, err = words.MembersOf(tx, "p1")

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	assertWords(t, wordsOfPage, "b")
}
