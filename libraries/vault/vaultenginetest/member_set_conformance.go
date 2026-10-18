package vaultenginetest

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const memberSetsBucket vault.Name = "member_sets"

var (
	theMember   = vault.StoredMember{ID: 0}
	aMember     = vault.StoredMember{ID: 7}
	zebraMember = vault.StoredMember{Raw: "zebra"}
)

func runMemberSetConformance(t *testing.T, open func(quotaBytes int64) (vault.Engine, error)) {
	t.Helper()

	t.Run(
		"MembersAddedAcrossTransactionsAreKept",
		func(t *testing.T) { membersAddedAcrossTransactionsAreKept(t, open) },
	)
	t.Run(
		"LaterChangesWinPerMember",
		func(t *testing.T) { laterChangesWinPerMember(t, open) },
	)
	t.Run(
		"ClearThenAddsGivesExactlyTheAdds",
		func(t *testing.T) { clearThenAddsGivesExactlyTheAdds(t, open) },
	)
	t.Run(
		"ClearedSetLeavesNoKey",
		func(t *testing.T) { clearedSetLeavesNoKey(t, open) },
	)
	t.Run(
		"MemberSetScanVisitsRangeInOrder",
		func(t *testing.T) { memberSetScanVisitsRangeInOrder(t, open) },
	)
	t.Run(
		"ConcurrentAddsToOneSetKeepEveryAdd",
		func(t *testing.T) { concurrentAddsToOneSetKeepEveryAdd(t, open) },
	)
	t.Run(
		"RemovingEveryRecordOfAGroupReadLeavesNoSet",
		func(t *testing.T) { removingEveryRecordOfAGroupReadLeavesNoSet(t, open) },
	)
}

func openMemberSets(t *testing.T, open func(int64) (vault.Engine, error)) vault.Engine {
	t.Helper()

	engine := openEngine(t, open, 0)
	if err := engine.ProvisionMemberSetsBucket(memberSetsBucket); err != nil {
		t.Fatalf("ProvisionMemberSetsBucket: %v", err)
	}

	return engine
}

func changeMemberSets(
	t *testing.T,
	engine vault.Engine,
	change func(vault.EngineMemberSets) error,
) {
	t.Helper()

	if err := engine.Update(context.Background(), func(etx vault.EngineTxn) error {
		return change(etx.MemberSets(memberSetsBucket))
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func memberSetKeyOf(group string) []byte {
	return stringKeyParts.Key(group).Bytes()
}

func adding(key string, members ...vault.StoredMember) func(vault.EngineMemberSets) error {
	return func(sets vault.EngineMemberSets) error {
		for _, member := range members {
			if err := sets.Add(memberSetKeyOf(key), member); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}
}

func storedMembersOf(t *testing.T, engine vault.Engine, key string) []vault.StoredMember {
	t.Helper()

	var stored []vault.StoredMember
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		members, err := etx.MemberSets(memberSetsBucket).Members(memberSetKeyOf(key))
		stored = members

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return sortedMembers(stored)
}

func sortedMembers(members []vault.StoredMember) []vault.StoredMember {
	return slices.SortedFunc(slices.Values(members), func(first, second vault.StoredMember) int {
		return cmp.Or(cmp.Compare(first.Raw, second.Raw), cmp.Compare(first.ID, second.ID))
	})
}

func assertMembers(t *testing.T, got []vault.StoredMember, want ...vault.StoredMember) {
	t.Helper()

	if got, want := sortedMembers(got), sortedMembers(want); !slices.Equal(got, want) {
		t.Fatalf("members = %v, want %v", got, want)
	}
}

func membersAddedAcrossTransactionsAreKept(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)

	changeMemberSets(t, engine, adding("p1", theMember, zebraMember))
	changeMemberSets(t, engine, adding("p1", aMember))

	assertMembers(t, storedMembersOf(t, engine, "p1"), theMember, aMember, zebraMember)
}

func laterChangesWinPerMember(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", theMember, aMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		key := memberSetKeyOf("p1")
		for _, step := range []func() error{
			func() error { return sets.Remove(key, theMember) },
			func() error { return sets.Add(key, theMember) },
			func() error { return sets.Add(key, zebraMember) },
			func() error { return sets.Remove(key, zebraMember) },
			func() error { return sets.Remove(key, aMember) },
		} {
			if err := step(); err != nil {
				return wrapTest(err)
			}
		}
		members, err := sets.Members(key)
		assertMembers(t, members, theMember)

		return err
	})

	assertMembers(t, storedMembersOf(t, engine, "p1"), theMember)
}

func clearThenAddsGivesExactlyTheAdds(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", theMember, aMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		if err := sets.Add(memberSetKeyOf("p1"), zebraMember); err != nil {
			return wrapTest(err)
		}
		if err := sets.Clear(memberSetKeyOf("p1")); err != nil {
			return wrapTest(err)
		}
		if err := adding("p1", aMember)(sets); err != nil {
			return err
		}
		members, err := sets.Members(memberSetKeyOf("p1"))
		assertMembers(t, members, aMember)

		return err
	})

	assertMembers(t, storedMembersOf(t, engine, "p1"), aMember)
}

func clearedSetLeavesNoKey(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", theMember))
	changeMemberSets(t, engine, adding("p2", aMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		if err := sets.Remove(memberSetKeyOf("p1"), zebraMember); err != nil {
			return wrapTest(err)
		}

		return sets.Clear(memberSetKeyOf("p1"))
	})

	keys := memberSetKeysWithin(t, engine, vault.EveryKey())
	if !slices.Equal(keys, []string{"p2"}) {
		t.Fatalf("sets = %v, want p2", keys)
	}
	assertMembers(t, storedMembersOf(t, engine, "p2"), aMember)
}

func memberSetKeysWithin(t *testing.T, engine vault.Engine, keys vault.KeyRange) []string {
	t.Helper()

	var visited []string
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		return etx.MemberSets(memberSetsBucket).Scan(
			keys,
			func(key []byte, _ []vault.StoredMember) (bool, error) {
				group, err := stringKeyParts.PartsOf(key)
				visited = append(visited, group)

				return true, err
			},
		)
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return visited
}

func memberSetScanVisitsRangeInOrder(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	for _, key := range []string{"c", "a", "b", "d"} {
		changeMemberSets(t, engine, adding(key, zebraMember))
	}

	got := memberSetKeysWithin(t, engine, stringKeyParts.KeysThroughFirst("c").After(
		stringKeyParts.Key("a"),
	))

	if !slices.Equal(got, []string{"b", "c"}) {
		t.Fatalf("scanned sets = %v, want b and c in order", got)
	}
}

func concurrentAddsToOneSetKeepEveryAdd(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)

	failures := make(chan error, concurrentWriters)
	var writing sync.WaitGroup
	for writer := range uint64(concurrentWriters) {
		writing.Go(func() {
			err := engine.Update(context.Background(), func(etx vault.EngineTxn) error {
				return etx.MemberSets(memberSetsBucket).Add(
					memberSetKeyOf("p1"),
					vault.StoredMember{ID: writer},
				)
			})
			failures <- err
		})
	}
	writing.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	if got := storedMembersOf(t, engine, "p1"); len(got) != concurrentWriters {
		t.Fatalf("members = %v, want %d", got, concurrentWriters)
	}
}

func removingEveryRecordOfAGroupReadLeavesNoSet(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	ctx := context.Background()
	engine := openEngine(t, open, 0)
	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	pages := registerPageWords(t, v)
	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return pages.mentions.PutEach(tx, mentionsOf("p2", "the", "zebra"))
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		words, err := pages.words.MembersOf(tx, "p2")
		if err != nil {
			return wrapTest(err)
		}
		for _, word := range words {
			if _, err := pages.mentions.Delete(tx, mention{word: word, page: "p2"}); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := engine.View(ctx, func(etx vault.EngineTxn) error {
		return etx.MemberSets(pageWordsBucket).Scan(
			vault.EveryKey(),
			func(key []byte, _ []vault.StoredMember) (bool, error) {
				return false, fmt.Errorf("set %q left, want none: %w", key, errAbortedOnPurpose)
			},
		)
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

const pageWordsBucket vault.Name = "words_of_pages"

type pageWords struct {
	mentions *vault.Collection[mention, string]
	words    *vault.MemberSetIndex[string, string, mention]
}

func registerPageWords(t *testing.T, v *vault.Vault) pageWords {
	t.Helper()

	ctx := context.Background()
	dictionary, err := v.RegisterFixedDictionary(ctx, "words", vault.TextKeyPart, []string{"the"})
	if err != nil {
		t.Fatalf("RegisterFixedDictionary: %v", err)
	}
	words, err := v.RegisterMemberSetIndex(
		pageWordsBucket,
		"1",
		vault.PairKey(vault.TextKeyPart, vault.TextKeyPart),
		func(stored mention) (string, string) { return stored.word, stored.page },
		func(word, page string) mention { return mention{word: word, page: page} },
		dictionary,
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterMemberSetIndex: %v", err)
	}
	mentions, err := v.RegisterIndexedCollection(
		ctx,
		vault.Name("page_mentions"),
		words,
		stringValueCodec{},
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterIndexedCollection: %v", err)
	}

	return pageWords{mentions: mentions, words: words}
}
