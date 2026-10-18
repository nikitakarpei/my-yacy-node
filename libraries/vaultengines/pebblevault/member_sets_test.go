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
	foldedSets                    = 4
	foldedTransactions            = 60
	rawMemberBytes                = 2 << 10
)

func openMemberSetEngine(t *testing.T, path string) *pebblevault.Engine {
	t.Helper()

	engine, err := pebblevault.OpenEngine(path, 0, testLimits, nil, nil)
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

func setKeyOf(set int) []byte {
	return []byte(fmt.Sprintf("p%d", set))
}

func changeBlindly(
	t *testing.T,
	engine *pebblevault.Engine,
) map[int][]vault.StoredMember {
	t.Helper()

	present := map[int]map[vault.StoredMember]bool{}
	for transaction := range foldedTransactions {
		set, member := transaction%foldedSets, rawMemberOf(transaction%17)
		isPresent := transaction%3 != 0
		if present[set] == nil {
			present[set] = map[vault.StoredMember]bool{}
		}
		present[set][member] = isPresent
		if err := engine.Update(context.Background(), func(etx vault.EngineTxn) error {
			sets := etx.MemberSets(memberSetsBucket)
			if isPresent {
				return sets.Add(setKeyOf(set), member)
			}

			return sets.Remove(setKeyOf(set), member)
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	return membersBySetFrom(present)
}

func membersBySetFrom(present map[int]map[vault.StoredMember]bool) map[int][]vault.StoredMember {
	membersBySet := map[int][]vault.StoredMember{}
	for set, presenceByMember := range present {
		for member, isPresent := range presenceByMember {
			if isPresent {
				membersBySet[set] = append(membersBySet[set], member)
			}
		}
		membersBySet[set] = sortedRaws(membersBySet[set])
	}

	return membersBySet
}

func sortedRaws(members []vault.StoredMember) []vault.StoredMember {
	return slices.SortedFunc(slices.Values(members), func(first, second vault.StoredMember) int {
		return strings.Compare(first.Raw, second.Raw)
	})
}

func storedMembersBySet(t *testing.T, engine *pebblevault.Engine) map[int][]vault.StoredMember {
	t.Helper()

	membersBySet := map[int][]vault.StoredMember{}
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		for set := range foldedSets {
			members, err := etx.MemberSets(memberSetsBucket).Members(setKeyOf(set))
			if err != nil {
				return err
			}
			membersBySet[set] = sortedRaws(members)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return membersBySet
}

func TestBlindMemberChangesSurviveReopening(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	before := openMemberSetEngine(t, path)
	want := changeBlindly(t, before)
	if err := before.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	after := openMemberSetEngine(t, path)
	t.Cleanup(func() {
		if err := after.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	got := storedMembersBySet(t, after)
	for set := range foldedSets {
		if !slices.Equal(got[set], want[set]) {
			t.Fatalf("members of set %d = %d, want %d", set, len(got[set]), len(want[set]))
		}
	}
}

func TestMemberSetsCountTowardUsedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	before := openMemberSetEngine(t, path)
	changeBlindly(t, before)
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
