package vaultenginetest

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func runPutEachConformance(t *testing.T, open func(quotaBytes int64) (vault.Engine, error)) {
	t.Helper()

	t.Run(
		"PutEachKeepsRecordCountsOfStoredAndRepeatedKeys",
		func(t *testing.T) { putEachKeepsRecordCountsOfStoredAndRepeatedKeys(t, open) },
	)
}

type mention struct {
	word string
	page string
}

func putEachKeepsRecordCountsOfStoredAndRepeatedKeys(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	v := openVault(t, open, 0)
	words, err := v.RegisterSecondPartIndex(
		vault.Name("mentions_by_page"),
		vault.PairKey(vault.TextKeyPart, vault.TextKeyPart),
		func(stored mention) (string, string) { return stored.word, stored.page },
		func(word, page string) mention { return mention{word: word, page: page} },
	)
	if err != nil {
		t.Fatalf("RegisterSecondPartIndex: %v", err)
	}
	mentions, err := v.RegisterIndexedCollection(
		context.Background(),
		vault.Name("mentions"),
		words,
		stringValueCodec{},
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterIndexedCollection: %v", err)
	}
	stored := mentionsOf("p1", "w00", "w01", "w02")
	written := append(mentionsOf("p1", numberedWords(12)...), mentionsOf("p1", "w05")...)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		if err := mentions.PutEach(tx, stored); err != nil {
			return err
		}

		return mentions.PutEach(tx, written)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	recordCounts, err := v.RecordCountsByBucket(context.Background())
	if err != nil {
		t.Fatalf("RecordCountsByBucket: %v", err)
	}
	if recordCounts["mentions"] != 12 || recordCounts["mentions_by_page"] != 12 {
		t.Fatalf("record counts = %v, want 12 mentions and 12 index entries", recordCounts)
	}
	if got := wordsOfPage(t, v, words, "p1"); !slices.Equal(got, numberedWords(12)) {
		t.Fatalf("words of p1 = %v, want %v", got, numberedWords(12))
	}
}

func mentionsOf(page string, words ...string) []vault.KeyedValue[mention, string] {
	keyedValues := make([]vault.KeyedValue[mention, string], 0, len(words))
	for _, word := range words {
		keyedValues = append(keyedValues, vault.KeyedValue[mention, string]{
			Key:   mention{word: word, page: page},
			Value: word,
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

func wordsOfPage(
	t *testing.T,
	v *vault.Vault,
	words *vault.SecondPartIndex[string, string, mention],
	page string,
) []string {
	t.Helper()

	var wordsOfPage []string
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		return words.Scan(tx, page, func(key mention) (bool, error) {
			wordsOfPage = append(wordsOfPage, key.word)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return wordsOfPage
}
