package peerdiscount_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/peerdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func documentOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func foundDocumentOf(
	t *testing.T,
	address string,
	placesGivenByPeers ...queryanswers.PlaceGivenByPeer,
) queryanswers.FoundDocument {
	t.Helper()

	return queryanswers.FoundDocument{
		Hash:               documentOf(t, address),
		Address:            address,
		Title:              "Berlin",
		PlacesGivenByPeers: placesGivenByPeers,
	}
}

func placeGivenBy(peer string, place int) queryanswers.PlaceGivenByPeer {
	return queryanswers.PlaceGivenByPeer{
		Peer:                 yacymodel.WordHash(peer),
		ReliabilityOfThePeer: 1,
		Place:                place,
	}
}

func addressesInOrderOf(
	orderedDocuments []queryanswers.FoundDocument,
) []string {
	addresses := make([]string, 0, len(orderedDocuments))
	for _, orderedDocument := range orderedDocuments {
		addresses = append(addresses, orderedDocument.Address)
	}

	return addresses
}

func orderedAddressesOf(
	t *testing.T,
	foundDocuments ...queryanswers.FoundDocument,
) []string {
	t.Helper()

	return addressesInOrderOf(peerdiscount.New().OrderedDocumentsOf(
		queryanswers.AnsweredQuery{FoundDocuments: foundDocuments},
	))
}

func TestADocumentSeveralPeersListedComesBeforeOneThatOnePeerListedFirst(t *testing.T) {
	t.Parallel()

	const listedByOnePeer, listedByThreePeers = "https://alone.example/", "https://agreed.example/"

	ordered := orderedAddressesOf(
		t,
		foundDocumentOf(t, listedByOnePeer, placeGivenBy("one", 0)),
		foundDocumentOf(
			t,
			listedByThreePeers,
			placeGivenBy("one", 3),
			placeGivenBy("two", 2),
			placeGivenBy("three", 1),
		),
	)

	if ordered[0] != listedByThreePeers {
		t.Errorf("the order reads %v, want the document three peers listed first", ordered)
	}
}

func TestOnePeerDoesNotHoldEveryPlaceOfTheOrder(t *testing.T) {
	t.Parallel()

	const listedByTheOtherPeer = "https://other.example/"

	foundDocuments := []queryanswers.FoundDocument{
		foundDocumentOf(t, "https://stuffed.example/one", placeGivenBy("one", 0)),
		foundDocumentOf(t, "https://stuffed.example/two", placeGivenBy("one", 1)),
		foundDocumentOf(t, "https://stuffed.example/three", placeGivenBy("one", 2)),
		foundDocumentOf(t, listedByTheOtherPeer, placeGivenBy("two", 9)),
	}

	ordered := orderedAddressesOf(t, foundDocuments...)

	if ordered[1] != listedByTheOtherPeer {
		t.Errorf(
			"the order reads %v, want the document of the other peer once one peer was placed",
			ordered,
		)
	}
}

func TestAReliablePeerCarriesMoreThanAPeerThisNodeBarelyKnows(t *testing.T) {
	t.Parallel()

	const listedByTheReliablePeer = "https://reliable.example/"

	ordered := orderedAddressesOf(
		t,
		queryanswers.FoundDocument{
			Hash:    documentOf(t, "https://barely-known.example/"),
			Address: "https://barely-known.example/",
			PlacesGivenByPeers: []queryanswers.PlaceGivenByPeer{{
				Peer: yacymodel.WordHash("two"), ReliabilityOfThePeer: 0.1, Place: 0,
			}},
		},
		queryanswers.FoundDocument{
			Hash:    documentOf(t, listedByTheReliablePeer),
			Address: listedByTheReliablePeer,
			PlacesGivenByPeers: []queryanswers.PlaceGivenByPeer{{
				Peer: yacymodel.WordHash("one"), ReliabilityOfThePeer: 1, Place: 0,
			}},
		},
	)

	if ordered[0] != listedByTheReliablePeer {
		t.Errorf("the order reads %v, want the document of the reliable peer first", ordered)
	}
}

func TestWhatAPeerCountsForADocumentDoesNotMoveIt(t *testing.T) {
	t.Parallel()

	const listedFirst, listedSecond = "https://first.example/", "https://second.example/"

	foundDocumentListedSecond := foundDocumentOf(t, listedSecond, placeGivenBy("one", 1))
	foundDocumentListedSecond.HitsPerQueryWord = map[yacymodel.Hash]int{
		yacymodel.WordHash("berlin"): 9000,
	}
	foundDocumentListedSecond.AmountOfWords = 9000
	foundDocumentListedSecond.Title = "Berlin Berlin Berlin"

	ordered := orderedAddressesOf(
		t,
		foundDocumentOf(t, listedFirst, placeGivenBy("one", 0)),
		foundDocumentListedSecond,
	)

	if ordered[0] != listedFirst {
		t.Errorf(
			"the order reads %v, want the order of the peer whatever the document counts",
			ordered,
		)
	}
}

func TestADocumentNoPeerListedComesLast(t *testing.T) {
	t.Parallel()

	const listedByNoPeer = "https://unlisted.example/"

	ordered := orderedAddressesOf(
		t,
		foundDocumentOf(t, listedByNoPeer),
		foundDocumentOf(t, "https://listed.example/", placeGivenBy("one", 9)),
	)

	if ordered[len(ordered)-1] != listedByNoPeer {
		t.Errorf("the order reads %v, want the document no peer listed last", ordered)
	}
}
