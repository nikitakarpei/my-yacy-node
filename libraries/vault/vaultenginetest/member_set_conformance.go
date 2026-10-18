package vaultenginetest

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const memberSetsBucket vault.Name = "member_sets"

var (
	idZeroMember  = vault.StoredMember{ID: 0}
	idSevenMember = vault.StoredMember{ID: 7}
	zebraMember   = vault.StoredMember{Raw: "zebra"}
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
		"AddsAfterADeleteAreTheWholeSet",
		func(t *testing.T) { addsAfterADeleteAreTheWholeSet(t, open) },
	)
	t.Run(
		"DeletedSetLeavesNoKey",
		func(t *testing.T) { deletedSetLeavesNoKey(t, open) },
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
		"RemovingTheLastMemberLeavesAnEmptySet",
		func(t *testing.T) { removingTheLastMemberLeavesAnEmptySet(t, open) },
	)
	t.Run(
		"RemovingFromAMissingSetLeavesAnEmptySet",
		func(t *testing.T) { removingFromAMissingSetLeavesAnEmptySet(t, open) },
	)
	t.Run(
		"ScanInAWriteSeesItsOwnChanges",
		func(t *testing.T) { scanInAWriteSeesItsOwnChanges(t, open) },
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

func memberSetKeyOf(setName string) []byte {
	return stringKeyParts.Key(setName).Bytes()
}

func adding(setName string, members ...vault.StoredMember) func(vault.EngineMemberSets) error {
	return func(sets vault.EngineMemberSets) error {
		for _, member := range members {
			if err := sets.Add(memberSetKeyOf(setName), member); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}
}

func storedMembersOf(t *testing.T, engine vault.Engine, setName string) []vault.StoredMember {
	t.Helper()

	var stored []vault.StoredMember
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		members, err := etx.MemberSets(memberSetsBucket).Members(memberSetKeyOf(setName))
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

	changeMemberSets(t, engine, adding("p1", idZeroMember, zebraMember))
	changeMemberSets(t, engine, adding("p1", idSevenMember))

	assertMembers(t, storedMembersOf(t, engine, "p1"), idZeroMember, idSevenMember, zebraMember)
}

func laterChangesWinPerMember(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", idZeroMember, idSevenMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		key := memberSetKeyOf("p1")
		for _, step := range []func() error{
			func() error { return sets.Remove(key, idZeroMember) },
			func() error { return sets.Add(key, idZeroMember) },
			func() error { return sets.Add(key, zebraMember) },
			func() error { return sets.Remove(key, zebraMember) },
			func() error { return sets.Remove(key, idSevenMember) },
		} {
			if err := step(); err != nil {
				return wrapTest(err)
			}
		}
		members, err := sets.Members(key)
		assertMembers(t, members, idZeroMember)

		return err
	})

	assertMembers(t, storedMembersOf(t, engine, "p1"), idZeroMember)
}

func addsAfterADeleteAreTheWholeSet(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", idZeroMember, idSevenMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		if err := sets.Add(memberSetKeyOf("p1"), zebraMember); err != nil {
			return wrapTest(err)
		}
		if err := sets.Delete(memberSetKeyOf("p1")); err != nil {
			return wrapTest(err)
		}
		if err := adding("p1", idSevenMember)(sets); err != nil {
			return err
		}
		members, err := sets.Members(memberSetKeyOf("p1"))
		assertMembers(t, members, idSevenMember)

		return err
	})

	assertMembers(t, storedMembersOf(t, engine, "p1"), idSevenMember)
}

func deletedSetLeavesNoKey(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", idZeroMember))
	changeMemberSets(t, engine, adding("p2", idSevenMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		if err := sets.Remove(memberSetKeyOf("p1"), zebraMember); err != nil {
			return wrapTest(err)
		}

		return sets.Delete(memberSetKeyOf("p1"))
	})

	keys := memberSetKeysWithin(t, engine, vault.EveryKey())
	if !slices.Equal(keys, []string{"p2"}) {
		t.Fatalf("sets = %v, want p2", keys)
	}
	assertMembers(t, storedMembersOf(t, engine, "p2"), idSevenMember)
}

func memberSetKeysWithin(t *testing.T, engine vault.Engine, keys vault.KeyRange) []string {
	t.Helper()

	var visited []string
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		return etx.MemberSets(memberSetsBucket).Scan(
			keys,
			func(key []byte) (bool, error) {
				setName, err := stringKeyParts.PartsOf(key)
				visited = append(visited, setName)

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
	for _, setName := range []string{"c", "a", "b", "d"} {
		changeMemberSets(t, engine, adding(setName, zebraMember))
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

func removingTheLastMemberLeavesAnEmptySet(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", idZeroMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		return sets.Remove(memberSetKeyOf("p1"), idZeroMember)
	})

	assertMembers(t, storedMembersOf(t, engine, "p1"))
	if keys := memberSetKeysWithin(
		t,
		engine,
		vault.EveryKey(),
	); !slices.Equal(
		keys,
		[]string{"p1"},
	) {
		t.Fatalf("sets = %v, want the emptied p1", keys)
	}
	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		return sets.Delete(memberSetKeyOf("p1"))
	})
	if keys := memberSetKeysWithin(t, engine, vault.EveryKey()); len(keys) != 0 {
		t.Fatalf("sets = %v after Delete, want none", keys)
	}
}

func removingFromAMissingSetLeavesAnEmptySet(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		return sets.Remove(memberSetKeyOf("p1"), idZeroMember)
	})

	assertMembers(t, storedMembersOf(t, engine, "p1"))
	if keys := memberSetKeysWithin(
		t,
		engine,
		vault.EveryKey(),
	); !slices.Equal(
		keys,
		[]string{"p1"},
	) {
		t.Fatalf("sets = %v, want the empty p1", keys)
	}
}

func scanInAWriteSeesItsOwnChanges(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openMemberSets(t, open)
	changeMemberSets(t, engine, adding("p1", idZeroMember))

	changeMemberSets(t, engine, func(sets vault.EngineMemberSets) error {
		if err := adding("p2", idSevenMember)(sets); err != nil {
			return err
		}
		if err := sets.Delete(memberSetKeyOf("p1")); err != nil {
			return wrapTest(err)
		}
		var visited []string
		err := sets.Scan(vault.EveryKey(), func(key []byte) (bool, error) {
			setName, err := stringKeyParts.PartsOf(key)
			visited = append(visited, setName)

			return true, err
		})
		if !slices.Equal(visited, []string{"p2"}) {
			t.Fatalf("sets = %v, want p2", visited)
		}

		return err
	})
}
