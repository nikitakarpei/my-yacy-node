package membersetfold_test

import (
	"math/rand/v2"
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
	return membersetfold.Change{Added: added, Removed: removed}.Encoded()
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
) membersetfold.Change {
	t.Helper()

	encoded, _, err := fold.Finish(includesBase)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	change, err := membersetfold.ChangeFrom(encoded)
	if err != nil {
		t.Fatalf("ChangeFrom: %v", err)
	}

	return change
}

func assertChange(
	t *testing.T,
	got membersetfold.Change,
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

func TestAFoldKeepsMembersOfEveryOperandInStoredOrder(t *testing.T) {
	fold := foldOf(t, operandOf([]vault.StoredMember{{Raw: "yak"}, idTwoMember}))

	if err := fold.MergeOlder(operandOf([]vault.StoredMember{zebraMember, {ID: 7}})); err != nil {
		t.Fatalf("MergeOlder: %v", err)
	}
	if err := fold.MergeNewer(
		operandOf([]vault.StoredMember{idOneMember, {Raw: "auk"}}),
	); err != nil {
		t.Fatalf("MergeNewer: %v", err)
	}

	assertChange(t, finishedChangeOf(t, fold, true), []vault.StoredMember{
		idOneMember, idTwoMember, {ID: 7}, {Raw: "auk"}, {Raw: "yak"}, zebraMember,
	}, nil)
}

func TestAFoldKeepsTheNewestChangeOfEveryMember(t *testing.T) {
	//nolint:gosec // G404: the test replays one fixed sequence.
	random := rand.New(rand.NewPCG(1, 2))
	for trial := range 2000 {
		operands := make([]membersetfold.Presence, 1+random.IntN(5))
		for index := range operands {
			operands[index] = randomPresence(random)
		}
		start := random.IntN(len(operands))
		fold := foldOf(t, operands[start].ChangeOf().Encoded())
		older, newer := start+1, start-1
		for older < len(operands) || newer >= 0 {
			var err error
			if newer >= 0 && (older == len(operands) || random.IntN(2) == 0) {
				err = fold.MergeNewer(operands[newer].ChangeOf().Encoded())
				newer--
			} else {
				err = fold.MergeOlder(operands[older].ChangeOf().Encoded())
				older++
			}
			if err != nil {
				t.Fatalf("trial %d: merge: %v", trial, err)
			}
		}
		includesBase := random.IntN(2) == 0

		want := newestPresenceAmong(operands)
		if includesBase {
			want = presentMembersOf(want)
		}
		assertChange(t, finishedChangeOf(t, fold, includesBase),
			sortedChangeOf(want).Added, sortedChangeOf(want).Removed)
	}
}

func randomPresence(random *rand.Rand) membersetfold.Presence {
	presence := membersetfold.Presence{}
	for range random.IntN(30) {
		member := vault.StoredMember{ID: random.Uint64N(40)}
		if random.IntN(2) == 0 {
			member = vault.StoredMember{Raw: string('a' + random.Int32N(20))}
		}
		presence[member] = random.IntN(3) != 0
	}

	return presence
}

func newestPresenceAmong(newestFirst []membersetfold.Presence) membersetfold.Presence {
	newest := membersetfold.Presence{}
	for _, presence := range newestFirst {
		for member, isPresent := range presence {
			if _, decided := newest[member]; !decided {
				newest[member] = isPresent
			}
		}
	}

	return newest
}

func presentMembersOf(presence membersetfold.Presence) membersetfold.Presence {
	present := membersetfold.Presence{}
	for member, isPresent := range presence {
		if isPresent {
			present[member] = true
		}
	}

	return present
}

func sortedChangeOf(presence membersetfold.Presence) membersetfold.Change {
	change, err := membersetfold.ChangeFrom(presence.ChangeOf().Encoded())
	if err != nil {
		panic(err)
	}

	return change
}
