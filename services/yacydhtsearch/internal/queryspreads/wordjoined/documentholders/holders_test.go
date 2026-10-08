package documentholders_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func documentHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("%q has no document hash: %v", address, err)
	}

	return hash
}

func answerOf(peerAddress string, documents ...yacymodel.URLHash) wordpartitionasks.ReplicaAnswer {
	answer := wordpartitionasks.ReplicaAnswer{
		Replica: peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash(peerAddress),
			Address: peerAddress,
		},
	}
	for _, document := range documents {
		answer.ListedDocuments = append(
			answer.ListedDocuments, wordpartitionasks.ListedDocument{Hash: document},
		)
	}

	return answer
}

func holdersAnswered(answers ...wordpartitionasks.ReplicaAnswer) documentholders.Holders {
	holders := documentholders.NoneYet()
	holders.WordPartitionAnswered(yacymodel.WordHash("berlin"), 0, answers)

	return holders
}

func withMetadataOnTheFirstDocument(
	answer wordpartitionasks.ReplicaAnswer,
) wordpartitionasks.ReplicaAnswer {
	answer.ListedDocuments[0] = wordpartitionasks.ListedDocumentFrom(
		yacymodel.URLMetadata{Hash: answer.ListedDocuments[0].Hash},
		yacymodel.None[yacymodel.RWIPosting](),
	)

	return answer
}

func TestTheDocumentFewerPeersHoldComesFirst(t *testing.T) {
	t.Parallel()

	heldByOne := documentHashOf(t, "https://held-by-one.example/")
	heldByTwo := documentHashOf(t, "https://held-by-two.example/")
	holders := holdersAnswered(
		answerOf("first", heldByTwo, heldByOne),
		answerOf("second", heldByTwo),
		answerOf("second", heldByTwo),
	).HoldersOf(yacymodel.URLHashes{heldByOne: {}, heldByTwo: {}})

	got := holders.LeastHeldFirst()
	if want := []yacymodel.URLHash{heldByOne, heldByTwo}; !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}

func TestDocumentsAsManyPeersHoldComeInTheirHashOrder(t *testing.T) {
	t.Parallel()

	first := documentHashOf(t, "https://first.example/")
	second := documentHashOf(t, "https://second.example/")
	holders := holdersAnswered(answerOf("peer", first, second)).
		HoldersOf(yacymodel.URLHashes{first: {}, second: {}})

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

	listed := documentHashOf(t, "https://listed.example/")
	other := documentHashOf(t, "https://other.example/")
	holders := holdersAnswered(
		answerOf("first", listed),
		answerOf("second", other),
		answerOf("third", listed, other),
	)

	got := sortedAddressesOf(holders.PeersHolding(listed))
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

	listed := documentHashOf(t, "https://listed.example/")
	unlisted := documentHashOf(t, "https://unlisted.example/")
	other := documentHashOf(t, "https://other.example/")
	holders := holdersAnswered(
		answerOf("first", listed, other),
		answerOf("second", listed),
		answerOf("third", other),
	).HoldersOf(yacymodel.URLHashes{listed: {}, unlisted: {}})

	got := sortedAddressesOf(holders.Peers())
	if want := []string{"first", "second"}; !slices.Equal(got, want) {
		t.Fatalf("the peers of the documents are %v, want %v", got, want)
	}
}

func TestOnlyADocumentNoAnswerCarriedMetadataForIsWithoutMetadata(t *testing.T) {
	t.Parallel()

	withMetadata := documentHashOf(t, "https://with-metadata.example/")
	withoutMetadata := documentHashOf(t, "https://without-metadata.example/")
	holders := holdersAnswered(
		withMetadataOnTheFirstDocument(answerOf("first", withMetadata, withoutMetadata)),
		answerOf("second", withMetadata),
	)

	got := holders.WithoutMetadataAmong(yacymodel.URLHashes{withMetadata: {}, withoutMetadata: {}})

	if want := (yacymodel.URLHashes{withoutMetadata: {}}); !maps.Equal(got, want) {
		t.Fatalf("the documents without metadata are %v, want %v", got, want)
	}
}
