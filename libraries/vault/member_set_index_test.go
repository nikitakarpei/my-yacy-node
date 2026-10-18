package vault_test

import (
	"context"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
)

const (
	pageWordsBucket    vault.Name = "words_of_pages"
	pageMentionsBucket vault.Name = "page_mentions"
)

type observedMemberSets struct {
	addedForms        []vault.MemberForm
	undecodableAmount int
}

func (o *observedMemberSets) ObserveMemberAdded(_ vault.Name, form vault.MemberForm) {
	o.addedForms = append(o.addedForms, form)
}

func (o *observedMemberSets) ObserveUndecodableMembers(_ vault.Name, amount int) {
	o.undecodableAmount += amount
}

type pageWords struct {
	engine     *doubleEngine
	vault      *vault.Vault
	collection *vault.Collection[mention, string]
	words      *vault.MemberSetIndex[string, string, mention]
	perPage    *vault.CountIndex[mention, string]
	observed   *observedMemberSets
}

func openPageWords(t *testing.T, engine *doubleEngine, valuesByID []string) pageWords {
	t.Helper()

	ctx := context.Background()
	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dictionary, err := vault.FixedDictionaryFrom(vault.TextKeyPart, valuesByID)
	if err != nil {
		t.Fatalf("FixedDictionaryFrom: %v", err)
	}
	observed := &observedMemberSets{}
	words, err := v.RegisterMemberSetIndex(
		pageWordsBucket,
		firstVersion,
		mentionKeyParts,
		func(stored mention) (string, string) { return stored.word, stored.page },
		func(word, page string) mention { return mention{word: word, page: page} },
		vault.MemberEncodingFrom(dictionary),
		observed,
	)
	if err != nil {
		t.Fatalf("RegisterMemberSetIndex: %v", err)
	}
	perPage, err := v.RegisterCountIndex(
		vault.Name("page_mention_counts"),
		firstVersion,
		stringKeyLayout,
		func(stored mention) string { return stored.page },
	)
	if err != nil {
		t.Fatalf("RegisterCountIndex: %v", err)
	}
	collection, err := v.RegisterIndexedCollection(
		ctx,
		pageMentionsBucket,
		words,
		stringValueCodec{},
		vault.ProjectedAs(func(key mention, _ string) mention { return key }, perPage),
	)
	if err != nil {
		t.Fatalf("RegisterIndexedCollection: %v", err)
	}

	return pageWords{
		engine:     engine,
		vault:      v,
		collection: collection,
		words:      words,
		perPage:    perPage,
		observed:   observed,
	}
}

func (p pageWords) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := p.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (p pageWords) mention(t *testing.T, page string, words ...string) {
	t.Helper()

	p.write(t, func(tx *vault.Txn) error {
		for _, word := range words {
			if _, err := p.collection.Put(tx, mention{word: word, page: page}, word); err != nil {
				return err
			}
		}

		return nil
	})
}

func (p pageWords) forget(tx *vault.Txn, page string, words ...string) error {
	for _, word := range words {
		if _, err := p.collection.Delete(tx, mention{word: word, page: page}); err != nil {
			return err
		}
	}

	return nil
}

func (p pageWords) wordsOf(t *testing.T, page string) []string {
	t.Helper()

	var words []string
	if err := p.vault.View(context.Background(), func(tx *vault.Txn) error {
		members, err := p.words.MembersOf(tx, page)
		words = members

		return err
	}); err != nil {
		t.Fatalf("MembersOf: %v", err)
	}
	slices.Sort(words)

	return words
}

func (p pageWords) storesSetOf(page string) bool {
	_, stored := p.engine.records[pageWordsBucket][string(stringKeyParts.Key(page).Bytes())]

	return stored
}

func assertWords(t *testing.T, got []string, want ...string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("words = %v, want %v", got, want)
	}
}

func TestAMemberSetHoldsTheWordsOfItsPage(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the", "a"})
	words.mention(t, "p1", "the", "zebra")
	words.mention(t, "p2", "a")

	assertWords(t, words.wordsOf(t, "p1"), "the", "zebra")
	assertWords(t, words.wordsOf(t, "p2"), "a")
}

func TestRemovingARecordRemovesItsMember(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p2", "the", "zebra", "yak")

	words.write(t, func(tx *vault.Txn) error { return words.forget(tx, "p2", "zebra") })

	assertWords(t, words.wordsOf(t, "p2"), "the", "yak")
}

func TestReplacingARecordKeepsItsMember(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the", "zebra")

	words.mention(t, "p1", "zebra")

	assertWords(t, words.wordsOf(t, "p1"), "the", "zebra")
}

func TestReplacingARecordAddsNoMember(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the", "zebra")

	words.mention(t, "p1", "zebra")

	if len(words.observed.addedForms) != 2 {
		t.Fatalf("added members = %d, want 2", len(words.observed.addedForms))
	}
}

func TestMembersInTheDictionaryAreStoredAsIDsAndOthersRaw(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the", "a"})

	words.mention(t, "p1", "a", "zebra")

	want := []vault.MemberForm{vault.MemberFormID, vault.MemberFormRaw}
	if !slices.Equal(words.observed.addedForms, want) {
		t.Fatalf("forms = %v, want %v", words.observed.addedForms, want)
	}
}

func TestATransactionReadsItsOwnMemberChanges(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the")

	words.write(t, func(tx *vault.Txn) error {
		if _, err := words.collection.Put(
			tx,
			mention{word: "zebra", page: "p1"},
			"zebra",
		); err != nil {
			return err
		}
		if err := words.forget(tx, "p1", "the"); err != nil {
			return err
		}
		members, err := words.words.MembersOf(tx, "p1")
		assertWords(t, members, "zebra")

		return err
	})
}

func TestRemovingEveryMemberReadDeletesTheSet(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the", "zebra")
	words.mention(t, "p2", "the")

	words.write(t, func(tx *vault.Txn) error {
		members, err := words.words.MembersOf(tx, "p1")
		if err != nil {
			return err
		}

		return words.forget(tx, "p1", members...)
	})

	if words.storesSetOf("p1") || !words.storesSetOf("p2") {
		t.Fatal("stored sets differ from p2 alone")
	}
}

func TestRemovingSomeMembersReadKeepsTheSet(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the", "zebra")

	words.write(t, func(tx *vault.Txn) error {
		if _, err := words.words.MembersOf(tx, "p1"); err != nil {
			return err
		}

		return words.forget(tx, "p1", "zebra")
	})

	assertWords(t, words.wordsOf(t, "p1"), "the")
}

func TestAddingAfterRemovingEveryMemberReadKeepsTheSet(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the")

	words.write(t, func(tx *vault.Txn) error {
		if _, err := words.words.MembersOf(tx, "p1"); err != nil {
			return err
		}
		if err := words.forget(tx, "p1", "the"); err != nil {
			return err
		}
		_, err := words.collection.Put(tx, mention{word: "zebra", page: "p1"}, "zebra")

		return err
	})

	assertWords(t, words.wordsOf(t, "p1"), "zebra")
}

func TestRemovingMembersWithoutReadingThemDeletesNoSet(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the")

	words.write(t, func(tx *vault.Txn) error { return words.forget(tx, "p1", "the") })

	if !words.storesSetOf("p1") {
		t.Fatal("set of p1 deleted without a read, want an empty set kept")
	}
}

func TestUndecodableMembersAreSkippedAndReported(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the")
	plantMembers(words.engine, "p1", vault.StoredMember{ID: 0}, vault.StoredMember{ID: 7},
		vault.StoredMember{Raw: "\x05"})

	assertWords(t, words.wordsOf(t, "p1"), "the")
	if words.observed.undecodableAmount != 2 {
		t.Fatalf("undecodable members = %d, want 2", words.observed.undecodableAmount)
	}
}

func plantMembers(engine *doubleEngine, page string, members ...vault.StoredMember) {
	engine.plant(
		pageWordsBucket,
		stringKeyParts.Key(page).Bytes(),
		memberset.AppendMembers(nil, members),
	)
}

func TestRemovingEveryDecodableMemberReadDeletesTheSet(t *testing.T) {
	words := openPageWords(t, newDoubleEngine(), []string{"the"})
	words.mention(t, "p1", "the")
	plantMembers(words.engine, "p1", vault.StoredMember{ID: 0}, vault.StoredMember{ID: 7})

	words.write(t, func(tx *vault.Txn) error {
		members, err := words.words.MembersOf(tx, "p1")
		if err != nil {
			return err
		}

		return words.forget(tx, "p1", members...)
	})

	if words.storesSetOf("p1") {
		t.Fatal("set of p1 kept, want deleted")
	}
}

func TestNewDictionaryValuesRebuildTheSets(t *testing.T) {
	engine := newDoubleEngine()
	before := openPageWords(t, engine, []string{"the", "a"})
	before.mention(t, "p1", "the", "zebra")
	before.mention(t, "p2", "a")

	after := openPageWords(t, engine, []string{"a", "the"})

	assertWords(t, after.wordsOf(t, "p1"), "the", "zebra")
	assertWords(t, after.wordsOf(t, "p2"), "a")
	if len(after.observed.addedForms) != 3 {
		t.Fatalf("members added at rebuild = %d, want 3", len(after.observed.addedForms))
	}
}

func TestUnchangedDictionaryValuesKeepTheSets(t *testing.T) {
	engine := newDoubleEngine()
	before := openPageWords(t, engine, []string{"the", "a"})
	before.mention(t, "p1", "the", "zebra")

	after := openPageWords(t, engine, []string{"the", "a"})

	assertWords(t, after.wordsOf(t, "p1"), "the", "zebra")
	if len(after.observed.addedForms) != 0 {
		t.Fatalf("members added at open = %d, want 0", len(after.observed.addedForms))
	}
}

func TestPutEachOverAMemberSetIndexStoresAsPutWould(t *testing.T) {
	valuesByID := []string{"w00", "w03"}
	byPut := openPageWords(t, newDoubleEngine(), valuesByID)
	byPutEach := openPageWords(t, newDoubleEngine(), valuesByID)
	stored := mentionsOf("p1", numberedWords(5)...)
	written := append(mentionsOf("p1", numberedWords(20)...), mentionsOf("p2", "w01")...)
	written = append(written, mentionsOf("p1", "w02")...)

	for _, words := range []pageWords{byPut, byPutEach} {
		words.write(t, func(tx *vault.Txn) error { return words.collection.PutEach(tx, stored) })
	}
	byPut.write(t, func(tx *vault.Txn) error {
		for _, keyedValue := range written {
			if _, err := byPut.collection.Put(tx, keyedValue.Key, keyedValue.Value); err != nil {
				return err
			}
		}

		return nil
	})
	byPutEach.write(
		t,
		func(tx *vault.Txn) error { return byPutEach.collection.PutEach(tx, written) },
	)

	for _, page := range []string{"p1", "p2"} {
		if want, got := byPut.countOf(t, page), byPutEach.countOf(t, page); got != want {
			t.Fatalf("records of %s after PutEach = %d, after Put %d", page, got, want)
		}
		wordsByPut, wordsByPutEach := byPut.wordsOf(t, page), byPutEach.wordsOf(t, page)
		if !slices.Equal(wordsByPutEach, wordsByPut) {
			t.Fatalf(
				"words of %s after PutEach = %v, after Put %v",
				page,
				wordsByPutEach,
				wordsByPut,
			)
		}
	}
}

func (p pageWords) countOf(t *testing.T, page string) int {
	t.Helper()

	var count int
	if err := p.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		count, err = p.perPage.Get(tx, page)

		return err
	}); err != nil {
		t.Fatalf("Get: %v", err)
	}

	return count
}
