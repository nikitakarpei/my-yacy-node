package scenarios_test

import (
	"crypto/ed25519"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/accountablepeer"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexhead"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

const english provensearch.Language = 1

type network struct {
	identity accountablepeer.Identity
	words    []indextree.Word
}

func networkOf(t *testing.T) network {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.NewChaCha8([32]byte{1}))
	if err != nil {
		t.Fatal(err)
	}
	words := make([]indextree.Word, 4)
	for index := range words {
		words[index] = indextree.Word{byte(index + 1), 0x5a}
	}

	return network{identity: accountablepeer.Identity{Network: "freeworld", Name: "peer-a", Key: key}, words: words}
}

func (n network) postings(seed uint64, amountPerWord int) []indextree.Leaf {
	random := rand.New(rand.NewPCG(seed, seed))
	var leaves []indextree.Leaf
	for _, word := range n.words {
		for range amountPerWord {
			var url indextree.URL
			for position := range url {
				url[position] = byte(random.Uint32())
			}
			leaves = append(leaves, indextree.Leaf{
				Key: indextree.KeyOf(word, url),
				Value: provensearch.PostingOf(provensearch.Posting{
					Hits:     uint16(random.IntN(500)),
					InTitle:  random.IntN(6) == 0,
					Day:      uint16(19000 + random.IntN(1000)),
					Language: provensearch.Language(1 + random.IntN(3)),
				}),
			})
		}
	}

	return leaves
}

func (n network) sealedPeer(leaves []indextree.Leaf, quota uint64) *accountablepeer.Peer {
	peer := accountablepeer.New(n.identity, quota)
	peer.Admit(leaves)
	peer.Seal()

	return peer
}

func verdictOf(t *testing.T, head indexhead.SignedHead, answer provensearch.Answer, query provensearch.Query, key ed25519.PublicKey) provensearch.Verdict {
	t.Helper()
	if !head.IsSignedBy(key) {
		t.Fatal("head is not signed by the peer")
	}
	verdict, err := provensearch.VerifiedVerdictOf(answer.Proof, query, head.Head.Root, head.Head.Summary)
	if err != nil {
		t.Fatalf("answer does not verify: %v", err)
	}

	return verdict
}

func TestAnHonestAnswerVerifiesAsComplete(t *testing.T) {
	network := networkOf(t)
	peer := network.sealedPeer(network.postings(1, 400), 1<<20)
	query := provensearch.Query{Words: network.words[:1], Amount: 10}
	head, answer := peer.Search(query, 400)
	if verdict := verdictOf(t, head, answer, query, peer.PublicKey()); !verdict.Complete || len(verdict.Results) != 10 {
		t.Fatal("honest answer does not verify as complete")
	}
}

func TestAnAnswerThatLowersAResultNoLongerLeadsToTheHead(t *testing.T) {
	network := networkOf(t)
	peer := network.sealedPeer(network.postings(2, 300), 1<<20)
	query := provensearch.Query{Words: network.words[:1], Amount: 5}
	head, answer := peer.Search(query, 400)
	hidden, _ := answer.Proof.Lookup(indextree.KeyOf(network.words[0], answer.Verdict.Results[0].URL))
	hidden.Revealed, hidden.Leaf, hidden.Summary = false, nil, indextree.Summary{Postings: 1}
	if _, err := provensearch.VerifiedVerdictOf(answer.Proof, query, head.Head.Root, head.Head.Summary); err == nil {
		t.Fatal("an answer that lowers its top result still verifies")
	}
}

func TestAnAnswerThatStopsEarlyIsPartialWithABoundAboveTheMissingResults(t *testing.T) {
	network := networkOf(t)
	peer := network.sealedPeer(network.postings(3, 300), 1<<20)
	query := provensearch.Query{Words: network.words[:1], Amount: 10}
	head, full := peer.Search(query, 400)
	_, early := peer.Search(query, 1)
	verdict := verdictOf(t, head, early, query, peer.PublicKey())
	if verdict.Complete {
		t.Fatal("an early stop verifies as complete")
	}
	for _, result := range full.Verdict.Results {
		if !slices.Contains(verdict.Results, result) && result.Score > verdict.Bound {
			t.Fatalf("missing result scores %d above the bound %d", result.Score, verdict.Bound)
		}
	}
}

func TestALanguageFilterIsCheckedAtTheRevealedPostings(t *testing.T) {
	network := networkOf(t)
	leaves := network.postings(4, 400)
	peer := network.sealedPeer(leaves, 1<<20)
	query := provensearch.Query{Words: network.words[:1], Amount: 10, Language: english}
	head, answer := peer.Search(query, 400)
	verdict := verdictOf(t, head, answer, query, peer.PublicKey())
	if !verdict.Complete || len(verdict.Results) != 10 {
		t.Fatal("a filtered answer is not complete")
	}
	if !slices.Equal(verdict.Results, expectedEnglishResults(leaves, network.words[0], head.Head.Summary.Postings)) {
		t.Fatal("a filtered answer holds other results than the filtered top 10")
	}
}

func expectedEnglishResults(leaves []indextree.Leaf, word indextree.Word, amountOfPostings uint64) []provensearch.Result {
	var amountOfWordPostings uint64
	for _, leaf := range leaves {
		if leaf.Key.Word() == word {
			amountOfWordPostings++
		}
	}
	weight := uint64(1)
	for ratio := amountOfPostings / amountOfWordPostings; ratio > 0; ratio >>= 1 {
		weight++
	}
	var results []provensearch.Result
	for _, leaf := range leaves {
		posting, _ := provensearch.PostingFrom(leaf.Value)
		if leaf.Key.Word() == word && posting.Language == english {
			results = append(results, provensearch.Result{URL: leaf.Key.URL(), Score: weight * uint64(provensearch.ImpactOf(posting))})
		}
	}
	slices.SortFunc(results, func(a, b provensearch.Result) int {
		if a.Score != b.Score {
			return int(b.Score) - int(a.Score)
		}

		return slices.Compare(a.URL[:], b.URL[:])
	})

	return results[:10]
}

func TestAHeadSignedByAnotherKeyIsRefused(t *testing.T) {
	network := networkOf(t)
	peer := network.sealedPeer(network.postings(5, 50), 1<<20)
	head, _ := peer.Search(provensearch.Query{Words: network.words[:1], Amount: 3}, 400)
	_, otherKey, _ := ed25519.GenerateKey(rand.NewChaCha8([32]byte{2}))
	forged := indexhead.SignedHeadOf(head.Head, otherKey)
	if forged.IsSignedBy(peer.PublicKey()) {
		t.Fatal("a head signed by another key passes as the peer's")
	}
}

func TestALookupProvesPresenceAndAbsence(t *testing.T) {
	network := networkOf(t)
	leaves := network.postings(6, 50)
	peer := network.sealedPeer(leaves[1:], 1<<20)
	head, proof := peer.Lookup([]indextree.Key{leaves[1].Key, leaves[0].Key})
	if err := provensearch.VerifiedProof(proof, head.Head.Root, head.Head.Summary); err != nil {
		t.Fatalf("lookup does not verify: %v", err)
	}
	if _, presence := proof.Lookup(leaves[1].Key); presence != indextree.Present {
		t.Fatal("a held posting is not proven present")
	}
	if _, presence := proof.Lookup(leaves[0].Key); presence != indextree.Absent {
		t.Fatal("a missing posting is not proven absent")
	}
}

func TestASealEvictsTheStalestURLsAboveTheQuota(t *testing.T) {
	network := networkOf(t)
	leaves := network.postings(7, 50)
	peer := network.sealedPeer(leaves, 150)
	head, proof := peer.Lookup(nil)
	if head.Head.Summary.Postings > 150 || provensearch.VerifiedProof(proof, head.Head.Root, head.Head.Summary) != nil {
		t.Fatalf("the sealed head holds %d postings above the quota", head.Head.Summary.Postings)
	}
}

func TestSearchesReadOnlyTheSealedTree(t *testing.T) {
	network := networkOf(t)
	leaves := network.postings(8, 20)
	peer := network.sealedPeer(leaves[:40], 1<<20)
	first, _ := peer.Lookup(nil)
	peer.Admit(leaves[40:])
	head, proof := peer.Lookup([]indextree.Key{leaves[50].Key})
	if _, presence := proof.Lookup(leaves[50].Key); presence != indextree.Absent || head.Head.Sequence != first.Head.Sequence {
		t.Fatal("an admitted posting is served before the seal")
	}
	if sealed := peer.Seal(); sealed.Head.Sequence <= first.Head.Sequence {
		t.Fatal("the head sequence does not increase")
	}
}
