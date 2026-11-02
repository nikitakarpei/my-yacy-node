package indextree_test

import (
	"math/rand/v2"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

func impactOfFirstByte(value []byte) indextree.Summary {
	return indextree.Summary{Postings: 1, TopImpact: uint16(value[0])}
}

func randomLeaves(seed uint64, amount, amountOfWords int) []indextree.Leaf {
	random := rand.New(rand.NewPCG(seed, seed))
	words := make([]indextree.Word, amountOfWords)
	for index := range words {
		for position := range words[index] {
			words[index][position] = byte(random.Uint32())
		}
	}
	leaves := make([]indextree.Leaf, 0, amount)
	for range amount {
		var url indextree.URL
		for position := range url {
			url[position] = byte(random.Uint32())
		}
		word := words[random.IntN(amountOfWords)]
		leaves = append(leaves, indextree.Leaf{
			Key:   indextree.KeyOf(word, url),
			Value: []byte{byte(random.Uint32()), byte(random.Uint32())},
		})
	}

	return leaves
}

func TestTheRootDependsOnlyOnTheStoredLeaves(t *testing.T) {
	leaves := randomLeaves(1, 500, 7)
	forward, _ := indextree.EmptyTree(impactOfFirstByte).Updated(indextree.Changes{Stored: leaves})
	reversed := make([]indextree.Leaf, len(leaves))
	for index, leaf := range leaves {
		reversed[len(leaves)-1-index] = leaf
	}
	backward := indextree.EmptyTree(impactOfFirstByte)
	for _, leaf := range reversed {
		backward, _ = backward.Updated(indextree.Changes{Stored: []indextree.Leaf{leaf}})
	}
	if forward.Root() != backward.Root() {
		t.Fatal("two insertion orders gave two roots")
	}
}

func TestRemovingLeavesRestoresTheEarlierRoot(t *testing.T) {
	leaves := randomLeaves(2, 300, 5)
	earlier, _ := indextree.EmptyTree(impactOfFirstByte).Updated(indextree.Changes{Stored: leaves[:200]})
	later, _ := earlier.Updated(indextree.Changes{Stored: leaves[200:]})
	removedKeys := make([]indextree.Key, 0, 100)
	for _, leaf := range leaves[200:] {
		removedKeys = append(removedKeys, leaf.Key)
	}
	restored, _ := later.Updated(indextree.Changes{Removed: removedKeys})
	if restored.Root() != earlier.Root() {
		t.Fatal("removing the added leaves did not restore the root")
	}
	if earlier.Summary().Postings != 200 {
		t.Fatalf("summary counts %d leaves, want 200", earlier.Summary().Postings)
	}
}

func TestALookupProofLeadsToTheRoot(t *testing.T) {
	leaves := randomLeaves(3, 400, 6)
	tree, _ := indextree.EmptyTree(impactOfFirstByte).Updated(indextree.Changes{Stored: leaves[:399]})
	prover := tree.Prover()
	prover.RevealLookupOf(leaves[10].Key)
	prover.RevealLookupOf(leaves[399].Key)
	proof := prover.Proof()
	root, summary, err := proof.VerifiedRoot(impactOfFirstByte)
	if err != nil || root != tree.Root() || summary != tree.Summary() {
		t.Fatalf("proof does not lead to the root: %v", err)
	}
	if found, presence := proof.Lookup(leaves[10].Key); presence != indextree.Present || found.Leaf == nil {
		t.Fatal("stored key is not proven present")
	}
	if _, presence := proof.Lookup(leaves[399].Key); presence != indextree.Absent {
		t.Fatal("missing key is not proven absent")
	}
}

func TestAChangedPostingBreaksTheProof(t *testing.T) {
	leaves := randomLeaves(4, 100, 3)
	tree, _ := indextree.EmptyTree(impactOfFirstByte).Updated(indextree.Changes{Stored: leaves})
	prover := tree.Prover()
	prover.RevealLookupOf(leaves[5].Key)
	proof := prover.Proof()
	found, _ := proof.Lookup(leaves[5].Key)
	found.Leaf.Value = []byte{0, 0}
	root, _, err := proof.VerifiedRoot(impactOfFirstByte)
	if err == nil && root == tree.Root() {
		t.Fatal("a changed posting still leads to the root")
	}
}

func TestAnUpdateWritesOnlyTheChangedPaths(t *testing.T) {
	leaves := randomLeaves(5, 1001, 10)
	tree, _ := indextree.EmptyTree(impactOfFirstByte).Updated(indextree.Changes{Stored: leaves[:1000]})
	_, amountOfWrittenNodes := tree.Updated(indextree.Changes{Stored: leaves[1000:]})
	if amountOfWrittenNodes < 2 || amountOfWrittenNodes > 40 {
		t.Fatalf("one stored leaf wrote %d nodes", amountOfWrittenNodes)
	}
}
