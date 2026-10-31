package documentholders_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

func answerOf(peerAddress string, documents ...yacymodel.URLHash) wordpartitionasks.ReplicaAnswer {
	answer := wordpartitionasks.ReplicaAnswer{
		Holder: peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash(peerAddress),
			Address: peerAddress,
		},
		ListedDocuments: yacymodel.URLHashesFrom(documents...),
	}

	return answer
}

func listingsAnswered(answers ...wordpartitionasks.ReplicaAnswer) *documentholders.Listings {
	listings := documentholders.NoListingsYet()
	listings.WordPartitionAnswered(yacymodel.WordHash("berlin"), 0, answers)

	return listings
}

func TestTheDocumentFewerPeersHoldComesFirst(t *testing.T) {
	t.Parallel()

	heldByOne := urlhashtest.URLHashOf(t, "https://held-by-one.example/")
	heldByTwo := urlhashtest.URLHashOf(t, "https://held-by-two.example/")
	holders := listingsAnswered(
		answerOf("first", heldByTwo, heldByOne),
		answerOf("second", heldByTwo),
		answerOf("second", heldByTwo),
	).HoldersOf(yacymodel.URLHashesFrom(heldByOne, heldByTwo))

	got := holders.LeastHeldFirst()
	if want := []yacymodel.URLHash{heldByOne, heldByTwo}; !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}

func TestDocumentsAsManyPeersHoldComeInTheirHashOrder(t *testing.T) {
	t.Parallel()

	first := urlhashtest.URLHashOf(t, "https://first.example/")
	second := urlhashtest.URLHashOf(t, "https://second.example/")
	holders := listingsAnswered(answerOf("peer", first, second)).
		HoldersOf(yacymodel.URLHashesFrom(first, second))

	got := holders.LeastHeldFirst()
	want := []yacymodel.URLHash{first, second}
	if second.String() < first.String() {
		want = []yacymodel.URLHash{second, first}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}

func TestThePeersHoldingADocumentAreThePeersThatListedIt(t *testing.T) {
	t.Parallel()

	listed := urlhashtest.URLHashOf(t, "https://listed.example/")
	other := urlhashtest.URLHashOf(t, "https://other.example/")
	holders := listingsAnswered(
		answerOf("first", listed),
		answerOf("second", other),
		answerOf("third", listed, other),
	)

	got := sortedAddressesOf(
		holders.HoldersOf(yacymodel.URLHashesFrom(listed)).PeersHolding(listed),
	)
	if want := []string{"first", "third"}; !slices.Equal(got, want) {
		t.Fatalf("the peers holding the document are %v, want %v", got, want)
	}
}

func sortedAddressesOf(peers []peerdirectory.AskablePeer) []string {
	addresses := make([]string, 0, len(peers))
	for _, peer := range peers {
		addresses = append(addresses, peer.Address)
	}
	slices.Sort(addresses)

	return addresses
}

func TestThePeersOfSomeDocumentsAreThePeersThatListedOneOfThem(t *testing.T) {
	t.Parallel()

	listed := urlhashtest.URLHashOf(t, "https://listed.example/")
	unlisted := urlhashtest.URLHashOf(t, "https://unlisted.example/")
	other := urlhashtest.URLHashOf(t, "https://other.example/")
	holders := listingsAnswered(
		answerOf("first", listed, other),
		answerOf("second", listed),
		answerOf("third", other),
	).HoldersOf(yacymodel.URLHashesFrom(listed, unlisted))

	got := sortedAddressesOf(holders.Peers())
	if want := []string{"first", "second"}; !slices.Equal(got, want) {
		t.Fatalf("the peers of the documents are %v, want %v", got, want)
	}
}
