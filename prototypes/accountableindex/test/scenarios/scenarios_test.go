package scenarios_test

import (
	"crypto/ed25519"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/accountablepeer"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexevidence"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexhead"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

const period = 10 * time.Minute

type fakeClock struct {
	moment time.Time
}

func (c *fakeClock) now() time.Time {
	return c.moment
}

func (c *fakeClock) advance() {
	c.moment = c.moment.Add(period)
}

type network struct {
	clock    *fakeClock
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

	return network{
		clock:    &fakeClock{moment: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)},
		identity: accountablepeer.Identity{Network: "freeworld", Name: "peer-a", Key: key},
		words:    words,
	}
}

func (n network) peer() *accountablepeer.Peer {
	return accountablepeer.New(n.identity, period, n.clock.now)
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
				Key:   indextree.KeyOf(word, url),
				Value: provensearch.PostingOf(uint16(random.IntN(500)), random.IntN(6) == 0),
			})
		}
	}

	return leaves
}

func (n network) sealed(peer *accountablepeer.Peer, leaves []indextree.Leaf) indexhead.SignedHead {
	peer.Accept(leaves)
	n.clock.advance()

	return peer.Seal()
}

func TestAnHonestAnswerVerifiesAsComplete(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	network.sealed(peer, network.postings(1, 400))
	query := provensearch.Query{Words: network.words[:2], Amount: 10}
	head, answer := peer.Search(query, 10000)
	verdict, err := provensearch.VerifiedVerdictOf(answer.Proof, query, head.Head.Root, head.Head.Summary)
	if err != nil || !head.IsSignedBy(peer.PublicKey()) || !verdict.Complete {
		t.Fatalf("honest answer does not verify as complete: %v", err)
	}
}

func TestAnAnswerThatDropsAResultNoLongerLeadsToTheHead(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	network.sealed(peer, network.postings(2, 300))
	query := provensearch.Query{Words: network.words[:1], Amount: 5}
	head, answer := peer.Search(query, 10000)
	top := answer.Verdict.Results[0]
	hidden, _ := answer.Proof.Lookup(indextree.KeyOf(network.words[0], top.URL))
	hidden.Revealed, hidden.Leaf, hidden.Summary = false, nil, indextree.Summary{Live: 1}
	if _, err := provensearch.VerifiedVerdictOf(answer.Proof, query, head.Head.Root, head.Head.Summary); err == nil {
		t.Fatal("an answer that lowers its top result still verifies")
	}
}

func TestAnAnswerThatStopsEarlyIsPartialWithABoundAboveTheMissingResults(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	network.sealed(peer, network.postings(3, 300))
	query := provensearch.Query{Words: network.words[:2], Amount: 10}
	head, full := peer.Search(query, 10000)
	_, early := peer.Search(query, 3)
	verdict, err := provensearch.VerifiedVerdictOf(early.Proof, query, head.Head.Root, head.Head.Summary)
	if err != nil || verdict.Complete {
		t.Fatalf("an early stop verifies as complete: %v", err)
	}
	for _, result := range full.Verdict.Results {
		found := false
		for _, shown := range verdict.Results {
			found = found || shown == result
		}
		if !found && result.Score > verdict.Bound {
			t.Fatalf("missing result scores %d above the bound %d", result.Score, verdict.Bound)
		}
	}
}

func TestAPeerThatDropsAcceptedPostingsBreaksItsPromise(t *testing.T) {
	network := networkOf(t)
	honest := network.peer()
	leaves := network.postings(4, 20)
	promise := honest.Accept(leaves)
	blackHole := network.peer()
	network.clock.advance()
	blackHole.Seal()
	observation := blackHole.Lookup(promise.Keys[:1])
	if _, broken := indexevidence.BrokenPromiseOf(promise, observation, honest.PublicKey()); !broken {
		t.Fatal("a dropped posting does not break the promise")
	}
	if _, broken := indexevidence.BrokenPromiseOf(promise, func() indexevidence.Observation {
		honest.Seal()

		return honest.Lookup(promise.Keys[:1])
	}(), honest.PublicKey()); broken {
		t.Fatal("a kept promise counts as broken")
	}
}

func TestAnEvictionLeavesATombstoneAndNoEvidence(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	leaves := network.postings(5, 50)
	network.sealed(peer, leaves)
	before := peer.Lookup([]indextree.Key{leaves[0].Key})
	peer.Remove([]indextree.Key{leaves[0].Key}, provensearch.RemovedByEviction)
	network.clock.advance()
	peer.Seal()
	after := peer.Lookup([]indextree.Key{leaves[0].Key})
	if _, undisclosed := indexevidence.UndisclosedRemovalOf(before, after, leaves[0].Key, peer.PublicKey()); undisclosed {
		t.Fatal("a disclosed eviction counts as undisclosed")
	}
	tombstone, _ := after.Proof.Lookup(leaves[0].Key)
	if _, isTombstone := provensearch.TombstoneEpochOf(tombstone.Leaf.Value); !isTombstone {
		t.Fatal("the evicted key holds no tombstone")
	}
}

func TestARemovalWithoutATombstoneIsEvidence(t *testing.T) {
	network := networkOf(t)
	leaves := network.postings(6, 50)
	tree, _ := indextree.EmptyTree(provensearch.SummaryOf).Updated(indextree.Changes{Stored: leaves})
	earlier := censorObservationOf(network, tree, 100, indextree.Hash{}, leaves[0].Key)
	censored, _ := tree.Updated(indextree.Changes{Removed: []indextree.Key{leaves[0].Key}})
	later := censorObservationOf(network, censored, 101, earlier.Head.Head.Hash(), leaves[0].Key)
	public, _ := network.identity.Key.Public().(ed25519.PublicKey)
	if _, undisclosed := indexevidence.UndisclosedRemovalOf(earlier, later, leaves[0].Key, public); !undisclosed {
		t.Fatal("a silent removal is not evidence")
	}
}

func censorObservationOf(
	network network,
	tree indextree.Tree,
	epoch uint64,
	previous indextree.Hash,
	key indextree.Key,
) indexevidence.Observation {
	prover := tree.Prover()
	prover.RevealLookupOf(key)
	head := indexhead.SignedHeadOf(indexhead.Head{
		Network: network.identity.Network, Peer: network.identity.Name, Epoch: epoch,
		Previous: previous, Root: tree.Root(), Summary: tree.Summary(),
	}, network.identity.Key)

	return indexevidence.Observation{Head: head, Proof: prover.Proof()}
}

func TestTwoStatesForOneEpochAreAFork(t *testing.T) {
	network := networkOf(t)
	first, second := network.peer(), network.peer()
	first.Accept(network.postings(7, 10))
	network.clock.advance()
	if _, isFork := indexhead.ForkOf(first.Seal(), second.Seal(), first.PublicKey()); !isFork {
		t.Fatal("two states for one epoch are not a fork")
	}
}

func TestACountIsCheckedByARandomRank(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	head := network.sealed(peer, network.postings(8, 120))
	observation := peer.CountProof(network.words[1], 77)
	err := provensearch.VerifiedProof(observation.Proof, head.Head.Root, head.Head.Summary)
	if err != nil || !observation.Proof.ReachesRankOf(network.words[1], 77) {
		t.Fatalf("a random rank does not check the count: %v", err)
	}
}

func TestAClockRunningBackSealsNoSecondHead(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	head := network.sealed(peer, network.postings(9, 10))
	network.clock.moment = network.clock.moment.Add(-3 * period)
	peer.Accept(network.postings(10, 10))
	if again := peer.Seal(); again.Head.Hash() != head.Head.Hash() {
		t.Fatal("a clock running back sealed a second head")
	}
}
