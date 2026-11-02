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
				Key: indextree.KeyOf(word, url),
				Value: provensearch.PostingOf(provensearch.Posting{
					Hits: uint16(random.IntN(500)), InTitle: random.IntN(6) == 0, Day: uint16(19000 + random.IntN(1000)),
				}),
			})
		}
	}

	return leaves
}

func (n network) sealed(peer *accountablepeer.Peer, leaves []indextree.Leaf) indexhead.SignedHead {
	peer.Admit(leaves)
	n.clock.advance()

	return peer.Seal()
}

func (n network) resealed(peer *accountablepeer.Peer) indexhead.SignedHead {
	n.clock.advance()

	return peer.Seal()
}

func TestAnHonestAnswerVerifiesAsComplete(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	network.sealed(peer, network.postings(1, 400))
	query := provensearch.Query{Words: network.words[:1], Amount: 10}
	head, answer := peer.Search(query, 400)
	verdict, err := provensearch.VerifiedVerdictOf(answer.Proof, query, head.Head.Root, head.Head.Summary)
	if err != nil || !head.IsSignedBy(peer.PublicKey()) || !verdict.Complete {
		t.Fatalf("honest answer does not verify as complete: %v", err)
	}
}

func TestAnAnswerThatLowersAResultNoLongerLeadsToTheHead(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	network.sealed(peer, network.postings(2, 300))
	query := provensearch.Query{Words: network.words[:1], Amount: 5}
	head, answer := peer.Search(query, 400)
	hidden, _ := answer.Proof.Lookup(indextree.KeyOf(network.words[0], answer.Verdict.Results[0].URL))
	hidden.Revealed, hidden.Leaf, hidden.Summary = false, nil, indextree.Summary{Live: 1}
	if _, err := provensearch.VerifiedVerdictOf(answer.Proof, query, head.Head.Root, head.Head.Summary); err == nil {
		t.Fatal("an answer that lowers its top result still verifies")
	}
}

func TestAnAnswerThatStopsEarlyIsPartialWithABoundAboveTheMissingResults(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	network.sealed(peer, network.postings(3, 300))
	query := provensearch.Query{Words: network.words[:1], Amount: 10}
	head, full := peer.Search(query, 400)
	_, early := peer.Search(query, 1)
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

func TestAPeerThatDropsAdmittedPostingsBreaksItsPromise(t *testing.T) {
	network := networkOf(t)
	honest := network.peer()
	promise := honest.Admit(network.postings(4, 20))
	blackHole := network.peer()
	network.clock.advance()
	blackHole.Seal()
	if _, broken := indexevidence.BrokenPromiseOf(promise, blackHole.Lookup(promise.Keys[:1]), honest.PublicKey()); !broken {
		t.Fatal("a dropped posting does not break the promise")
	}
	honest.Seal()
	if _, broken := indexevidence.BrokenPromiseOf(promise, honest.Lookup(promise.Keys), honest.PublicKey()); broken {
		t.Fatal("a kept promise counts as broken")
	}
}

func TestAPromiseIsKeptByADisclosedRemoval(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	leaves := network.postings(5, 20)
	promise := peer.Admit(leaves)
	handedOff := leaves[len(leaves)-1].Key
	peer.RemoveURL(leaves[0].Key.URL(), provensearch.RemovedByNewerVersion)
	peer.HandOff(handedOff.Word(), provensearch.PartitionOf(handedOff.URL()), []string{"peer-b", "peer-c"})
	network.clock.advance()
	peer.Seal()
	if _, broken := indexevidence.BrokenPromiseOf(promise, peer.Lookup(promise.Keys), peer.PublicKey()); broken {
		t.Fatal("a disclosed removal breaks the promise")
	}
}

func TestAnEvictionLeavesATombstoneAndNoEvidence(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	leaves := network.postings(6, 50)
	network.sealed(peer, leaves)
	before := peer.Lookup([]indextree.Key{leaves[0].Key})
	peer.RemoveURL(leaves[0].Key.URL(), provensearch.RemovedByEviction)
	network.resealed(peer)
	after := peer.Lookup([]indextree.Key{leaves[0].Key})
	if _, undisclosed := indexevidence.UndisclosedRemovalOf(before, after, leaves[0].Key, peer.PublicKey()); undisclosed {
		t.Fatal("a disclosed eviction counts as undisclosed")
	}
}

func TestAResetDisclosesEveryEarlierRemoval(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	leaves := network.postings(13, 30)
	promise := peer.Admit(leaves)
	network.clock.advance()
	peer.Seal()
	before := peer.Lookup([]indextree.Key{leaves[0].Key})
	peer.Reset()
	network.resealed(peer)
	after := peer.Lookup([]indextree.Key{leaves[0].Key})
	if _, undisclosed := indexevidence.UndisclosedRemovalOf(before, after, leaves[0].Key, peer.PublicKey()); undisclosed {
		t.Fatal("a removal by reset counts as undisclosed")
	}
	if _, broken := indexevidence.BrokenPromiseOf(promise, after, peer.PublicKey()); broken {
		t.Fatal("a reset breaks an earlier promise")
	}
}

func TestATombstoneOutlivesAPauseInSealing(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	leaves := network.postings(14, 30)
	network.sealed(peer, leaves)
	peer.RemoveURL(leaves[0].Key.URL(), provensearch.RemovedByOperator)
	network.clock.moment = network.clock.moment.Add((indexevidence.TombstoneEpochs + 10) * period)
	peer.Seal()
	shown := peer.Lookup([]indextree.Key{leaves[0].Key})
	if _, presence := shown.Proof.Lookup(provensearch.RemovedURLKeyOf(leaves[0].Key.URL())); presence != indextree.Present {
		t.Fatal("a tombstone was dropped before any head showed it")
	}
}

func TestARemovalWithoutATombstoneIsEvidence(t *testing.T) {
	network := networkOf(t)
	leaves := network.postings(7, 50)
	tree, _ := indextree.EmptyTree(provensearch.SummaryOf).Updated(indextree.Changes{Stored: leaves})
	earlier := censorObservationOf(network, tree, indexhead.Head{Epoch: 100}, leaves[0].Key)
	censored, _ := tree.Updated(indextree.Changes{Removed: []indextree.Key{leaves[0].Key}})
	later := censorObservationOf(network, censored, indexhead.Head{
		Epoch: 101, PredecessorEpoch: 100, Predecessor: earlier.Head.Head.Hash(),
	}, leaves[0].Key)
	if _, undisclosed := indexevidence.UndisclosedRemovalOf(earlier, later, leaves[0].Key, publicKeyOf(network)); !undisclosed {
		t.Fatal("a silent removal is not evidence")
	}
}

func censorObservationOf(network network, tree indextree.Tree, head indexhead.Head, key indextree.Key) indexevidence.Observation {
	prover := tree.Prover()
	prover.RevealLookupOf(key)
	for _, tombstoneKey := range indexevidence.TombstoneKeysOf(key) {
		prover.RevealLookupOf(tombstoneKey)
	}
	head.Network, head.Peer = network.identity.Network, network.identity.Name
	head.Root, head.Summary = tree.Root(), tree.Summary()

	return indexevidence.Observation{Head: indexhead.SignedHeadOf(head, network.identity.Key), Proof: prover.Proof()}
}

func publicKeyOf(network network) ed25519.PublicKey {
	public, _ := network.identity.Key.Public().(ed25519.PublicKey)

	return public
}

func TestTwoStatesForOneEpochAreAFork(t *testing.T) {
	network := networkOf(t)
	first, second := network.peer(), network.peer()
	first.Admit(network.postings(10, 10))
	network.clock.advance()
	if _, isFork := indexhead.ForkOf(first.Seal(), second.Seal(), first.PublicKey()); !isFork {
		t.Fatal("two states for one epoch are not a fork")
	}
}

func TestAChainThatSkipsAnotherHeadIsAFork(t *testing.T) {
	network := networkOf(t)
	first, second := network.peer(), network.peer()
	network.clock.advance()
	first.Seal()
	network.clock.advance()
	shown := first.Seal()
	network.clock.advance()
	network.clock.advance()
	hidden := second.Seal()
	if _, isFork := indexhead.ForkOf(shown, hidden, first.PublicKey()); !isFork {
		t.Fatal("a second chain that skips a shown head is not a fork")
	}
	following := network.resealed(first)
	if _, isFork := indexhead.ForkOf(shown, following, first.PublicKey()); isFork {
		t.Fatal("one honest chain counts as a fork")
	}
}

func TestAClockRunningBackSealsNoSecondHead(t *testing.T) {
	network := networkOf(t)
	peer := network.peer()
	head := network.sealed(peer, network.postings(11, 10))
	network.clock.moment = network.clock.moment.Add(-3 * period)
	peer.Admit(network.postings(12, 10))
	if again := peer.Seal(); again.Head.Hash() != head.Head.Hash() {
		t.Fatal("a clock running back sealed a second head")
	}
}
