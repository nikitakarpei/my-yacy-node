package documentholders_test

import (
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

func TestTheDocumentMorePeersHoldComesFirst(t *testing.T) {
	t.Parallel()

	heldByOne := documentHashOf(t, "https://held-by-one.example/")
	heldByTwo := documentHashOf(t, "https://held-by-two.example/")
	holders := documentholders.Holders{}

	holders.AddHoldersIn([]wordpartitionasks.ReplicaAnswer{
		answerOf("first", heldByOne, heldByTwo),
		answerOf("second", heldByTwo),
		answerOf("second", heldByTwo),
	})

	got := holders.MostHeldFirst(yacymodel.URLHashes{heldByOne: {}, heldByTwo: {}})
	if want := []yacymodel.URLHash{heldByTwo, heldByOne}; !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}

func TestDocumentsAsManyPeersHoldComeInTheirHashOrder(t *testing.T) {
	t.Parallel()

	first := documentHashOf(t, "https://first.example/")
	second := documentHashOf(t, "https://second.example/")
	holders := documentholders.Holders{}

	holders.AddHoldersIn([]wordpartitionasks.ReplicaAnswer{answerOf("peer", first, second)})

	got := holders.MostHeldFirst(yacymodel.URLHashes{first: {}, second: {}})
	want := []yacymodel.URLHash{first, second}
	if second.String() < first.String() {
		want = []yacymodel.URLHash{second, first}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}
