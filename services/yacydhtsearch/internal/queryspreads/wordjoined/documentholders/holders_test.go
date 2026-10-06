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
	answer.ListedDocuments[0].Metadata = yacymodel.Some(
		yacymodel.URLMetadata{Hash: answer.ListedDocuments[0].Hash},
	)

	return answer
}

func TestTheDocumentMorePeersHoldComesFirst(t *testing.T) {
	t.Parallel()

	heldByOne := documentHashOf(t, "https://held-by-one.example/")
	heldByTwo := documentHashOf(t, "https://held-by-two.example/")
	holders := holdersAnswered(
		answerOf("first", heldByOne, heldByTwo),
		answerOf("second", heldByTwo),
		answerOf("second", heldByTwo),
	).HoldersOf(yacymodel.URLHashes{heldByOne: {}, heldByTwo: {}})

	got := holders.MostHeldFirst()
	if want := []yacymodel.URLHash{heldByTwo, heldByOne}; !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}

func TestDocumentsAsManyPeersHoldComeInTheirHashOrder(t *testing.T) {
	t.Parallel()

	first := documentHashOf(t, "https://first.example/")
	second := documentHashOf(t, "https://second.example/")
	holders := holdersAnswered(answerOf("peer", first, second)).
		HoldersOf(yacymodel.URLHashes{first: {}, second: {}})

	got := holders.MostHeldFirst()
	want := []yacymodel.URLHash{first, second}
	if second.String() < first.String() {
		want = []yacymodel.URLHash{second, first}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the holders ordered %v, want %v", got, want)
	}
}

func TestThePeersWithTheirDocumentsComeInTheOrderOfTheirHashes(t *testing.T) {
	t.Parallel()

	first := documentHashOf(t, "https://first.example/")
	second := documentHashOf(t, "https://second.example/")
	holders := holdersAnswered(
		answerOf("zulu", first),
		answerOf("alpha", second),
		answerOf("zulu", second),
	).HoldersOf(yacymodel.URLHashes{first: {}, second: {}})

	got := holders.PeersWithTheirDocuments()
	wantPeers := []string{"zulu", "alpha"}
	if yacymodel.WordHash("alpha").String() < yacymodel.WordHash("zulu").String() {
		wantPeers = []string{"alpha", "zulu"}
	}
	wantDocuments := map[string]yacymodel.URLHashes{
		"zulu":  {first: {}, second: {}},
		"alpha": {second: {}},
	}
	if len(got) != len(wantPeers) {
		t.Fatalf("the holders hold %v, want the peers %v", got, wantPeers)
	}
	for place, peer := range got {
		if peer.Peer.Address != wantPeers[place] ||
			!maps.Equal(peer.Documents, wantDocuments[peer.Peer.Address]) {
			t.Fatalf(
				"the holders hold %v, want the peers %v with %v",
				got,
				wantPeers,
				wantDocuments,
			)
		}
	}
}

func TestTheHoldersOfSomeDocumentsHoldOnlyThoseThePeersListed(t *testing.T) {
	t.Parallel()

	listed := documentHashOf(t, "https://listed.example/")
	unlisted := documentHashOf(t, "https://unlisted.example/")
	other := documentHashOf(t, "https://other.example/")
	holders := holdersAnswered(answerOf("first", listed, other), answerOf("second", listed)).
		HoldersOf(yacymodel.URLHashes{listed: {}, unlisted: {}})

	got := holders.PeersWithTheirDocuments()
	want := map[string]yacymodel.URLHashes{"first": {listed: {}}, "second": {listed: {}}}
	if len(got) != len(want) {
		t.Fatalf("the holders hold %v, want %v", got, want)
	}
	for _, peer := range got {
		if !maps.Equal(peer.Documents, want[peer.Peer.Address]) {
			t.Fatalf("the holders hold %v, want %v", got, want)
		}
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
