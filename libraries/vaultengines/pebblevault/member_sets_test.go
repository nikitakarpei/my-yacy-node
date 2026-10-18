package pebblevault_test

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const (
	memberSetsBucket   vault.Name = "words_of_pages"
	foldedGroups                  = 4
	foldedTransactions            = 300
	rawMemberBytes                = 2 << 10
)

var flushingLimits = pebblevault.MachineLimits{
	BlockCacheBytes:  8 << 20,
	MemtableBytes:    256 << 10,
	WriteConcurrency: 2,
}

func openMemberSetEngine(t *testing.T, path string) *pebblevault.Engine {
	t.Helper()

	engine, err := pebblevault.OpenEngine(path, 0, flushingLimits, nil, nil)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	if err := engine.ProvisionMemberSetsBucket(memberSetsBucket); err != nil {
		t.Fatalf("ProvisionMemberSetsBucket: %v", err)
	}

	return engine
}

func rawMemberOf(number int) vault.StoredMember {
	return vault.StoredMember{
		Raw: fmt.Sprintf("%04d", number) + strings.Repeat("x", rawMemberBytes),
	}
}

func groupKeyOf(group int) []byte {
	return []byte(fmt.Sprintf("p%d", group))
}

func foldMany(t *testing.T, engine *pebblevault.Engine) map[int][]vault.StoredMember {
	t.Helper()

	present := map[int]map[vault.StoredMember]bool{}
	for transaction := range foldedTransactions {
		group, member := transaction%foldedGroups, rawMemberOf(transaction%17)
		isPresent := transaction%3 != 0
		if present[group] == nil {
			present[group] = map[vault.StoredMember]bool{}
		}
		present[group][member] = isPresent
		if err := engine.Update(context.Background(), func(etx vault.EngineTxn) error {
			sets := etx.MemberSets(memberSetsBucket)
			if isPresent {
				return sets.Add(groupKeyOf(group), member)
			}

			return sets.Remove(groupKeyOf(group), member)
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	return membersByGroupFrom(present)
}

func membersByGroupFrom(present map[int]map[vault.StoredMember]bool) map[int][]vault.StoredMember {
	membersByGroup := map[int][]vault.StoredMember{}
	for group, presenceByMember := range present {
		for member, isPresent := range presenceByMember {
			if isPresent {
				membersByGroup[group] = append(membersByGroup[group], member)
			}
		}
		membersByGroup[group] = sortedRaws(membersByGroup[group])
	}

	return membersByGroup
}

func sortedRaws(members []vault.StoredMember) []vault.StoredMember {
	return slices.SortedFunc(slices.Values(members), func(first, second vault.StoredMember) int {
		return strings.Compare(first.Raw, second.Raw)
	})
}

func storedMembersByGroup(t *testing.T, engine *pebblevault.Engine) map[int][]vault.StoredMember {
	t.Helper()

	membersByGroup := map[int][]vault.StoredMember{}
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		for group := range foldedGroups {
			members, err := etx.MemberSets(memberSetsBucket).Members(groupKeyOf(group))
			if err != nil {
				return err
			}
			membersByGroup[group] = sortedRaws(members)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return membersByGroup
}

func TestMemberSetsFoldManyBlindChangesAcrossFlushesAndReopening(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	before := openMemberSetEngine(t, path)
	want := foldMany(t, before)
	if err := before.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	after := openMemberSetEngine(t, path)
	t.Cleanup(func() {
		if err := after.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	got := storedMembersByGroup(t, after)
	for group := range foldedGroups {
		if !slices.Equal(got[group], want[group]) {
			t.Fatalf("members of group %d = %d, want %d", group, len(got[group]), len(want[group]))
		}
	}
}

func TestMemberSetsCountTowardUsedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	before := openMemberSetEngine(t, path)
	foldMany(t, before)
	if err := before.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	after := openMemberSetEngine(t, path)
	t.Cleanup(func() {
		if err := after.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	used, err := after.UsedBytes(context.Background())
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	if used < rawMemberBytes {
		t.Fatalf("used bytes = %d, want at least one member's %d", used, rawMemberBytes)
	}
}
