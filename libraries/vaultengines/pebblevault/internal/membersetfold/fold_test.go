package membersetfold_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/internal/membersetfold"
)

var (
	idOneMember   = vault.StoredMember{ID: 1}
	idTwoMember   = vault.StoredMember{ID: 2}
	zebraMember   = vault.StoredMember{Raw: "zebra"}
	unreadableSet = []byte{0x7f}
)

func operandOf(added []vault.StoredMember, removed ...vault.StoredMember) []byte {
	return vault.MemberSetChange{Added: added, Removed: removed}.Encoded()
}

func foldOf(t *testing.T, newest []byte) *membersetfold.Fold {
	t.Helper()

	fold, err := membersetfold.From(newest)
	if err != nil {
		t.Fatalf("From: %v", err)
	}

	return fold
}

func finishedChangeOf(
	t *testing.T,
	fold *membersetfold.Fold,
	includesBase bool,
) vault.MemberSetChange {
	t.Helper()

	encoded, _, err := fold.Finish(includesBase)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	change, err := vault.MemberSetChangeFrom(encoded)
	if err != nil {
		t.Fatalf("MemberSetChangeFrom: %v", err)
	}

	return change
}

func assertChange(
	t *testing.T,
	got vault.MemberSetChange,
	wantAdded []vault.StoredMember,
	wantRemoved []vault.StoredMember,
) {
	t.Helper()

	if !slices.Equal(got.Added, wantAdded) || !slices.Equal(got.Removed, wantRemoved) {
		t.Fatalf("change = %+v, want added %v and removed %v", got, wantAdded, wantRemoved)
	}
}

func TestANewerChangeWinsOverAnOlderOne(t *testing.T) {
	fold := foldOf(t, operandOf([]vault.StoredMember{idOneMember}))

	if err := fold.MergeNewer(operandOf(nil, idOneMember)); err != nil {
		t.Fatalf("MergeNewer: %v", err)
	}
	if err := fold.MergeOlder(
		operandOf([]vault.StoredMember{idOneMember, idTwoMember}),
	); err != nil {
		t.Fatalf("MergeOlder: %v", err)
	}

	assertChange(t, finishedChangeOf(t, fold, false), []vault.StoredMember{idTwoMember},
		[]vault.StoredMember{idOneMember})
}

func TestAPartialFoldKeepsItsRemoves(t *testing.T) {
	fold := foldOf(t, operandOf([]vault.StoredMember{zebraMember}, idOneMember))

	assertChange(t, finishedChangeOf(t, fold, false), []vault.StoredMember{zebraMember},
		[]vault.StoredMember{idOneMember})
}

func TestAFoldThatReachesTheBaseKeepsOnlyPresentMembers(t *testing.T) {
	fold := foldOf(t, operandOf(nil, idTwoMember))

	if err := fold.MergeOlder(
		operandOf([]vault.StoredMember{idOneMember, idTwoMember}),
	); err != nil {
		t.Fatalf("MergeOlder: %v", err)
	}

	assertChange(t, finishedChangeOf(t, fold, true), []vault.StoredMember{idOneMember}, nil)
}

func TestAFoldRefusesAnUnreadableOperand(t *testing.T) {
	if _, err := membersetfold.From(unreadableSet); err == nil {
		t.Fatal("From an unreadable operand succeeded, want refusal")
	}
	fold := foldOf(t, operandOf([]vault.StoredMember{idOneMember}))
	if err := fold.MergeNewer(unreadableSet); err == nil {
		t.Fatal("MergeNewer of an unreadable operand succeeded, want refusal")
	}
	if err := fold.MergeOlder(unreadableSet); err == nil {
		t.Fatal("MergeOlder of an unreadable operand succeeded, want refusal")
	}
}
