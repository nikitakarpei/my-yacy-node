// Package peerdiscount orders the found documents by the agreement of the
// peers about them, which falls by half for each document of the same peer it
// already placed above. It reads nothing a peer said about the contents of a
// document: a document stands on how many peers listed it, how high each of
// them listed it, and how reliable this node found those peers. A peer this
// node found unreliable still counts, at half the weight of one it found wholly
// reliable, so a network this node has just met still orders its documents. One
// peer thus holds the whole first page only while no other peer listed
// anything.
// Documents of equal discounted agreement keep the order of falling agreement,
// in which documents of equal agreement keep the order the spread found them
// in.
package peerdiscount

import (
	"cmp"
	"math"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shareOfAgreementKeptPerPlacedDocumentOfTheSamePeer = 0.5
	saturationOfThePlaceInOneList                      = 10.0
	weightOfAPeerThisNodeFoundUnreliable               = 1.0
)

type Ordering struct{}

func New() Ordering {
	return Ordering{}
}

func (Ordering) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	agreementPerDocument := agreementPerDocumentOf(answers.FoundDocuments)

	return documentsInFallingOrderOfDiscountedAgreement(
		documentsInFallingOrderOfAgreement(
			slices.Clone(answers.FoundDocuments), agreementPerDocument,
		),
		agreementPerDocument,
	)
}

func agreementPerDocumentOf(
	foundDocuments []queryanswers.FoundDocument,
) map[yacymodel.URLHash]float64 {
	agreementPerDocument := make(map[yacymodel.URLHash]float64, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		agreementPerDocument[foundDocument.Hash] = agreementOf(foundDocument)
	}

	return agreementPerDocument
}

func agreementOf(foundDocument queryanswers.FoundDocument) float64 {
	agreement := 0.0
	for _, placeGivenByPeer := range foundDocument.PlacesGivenByPeers {
		agreement += weightOfThePeer(placeGivenByPeer.ReliabilityOfThePeer) /
			(saturationOfThePlaceInOneList + float64(placeGivenByPeer.Place) + 1)
	}

	return agreement
}

func weightOfThePeer(reliabilityOfThePeer float64) float64 {
	return weightOfAPeerThisNodeFoundUnreliable + reliabilityOfThePeer
}

func documentsInFallingOrderOfAgreement(
	foundDocuments []queryanswers.FoundDocument,
	agreementPerDocument map[yacymodel.URLHash]float64,
) []queryanswers.FoundDocument {
	slices.SortStableFunc(foundDocuments, func(one, other queryanswers.FoundDocument) int {
		return cmp.Compare(
			agreementPerDocument[other.Hash], agreementPerDocument[one.Hash],
		)
	})

	return foundDocuments
}

func documentsInFallingOrderOfDiscountedAgreement(
	documentsOfFallingAgreement []queryanswers.FoundDocument,
	agreementPerDocument map[yacymodel.URLHash]float64,
) []queryanswers.FoundDocument {
	unplacedDocuments := documentsListedByPeersOf(documentsOfFallingAgreement)
	amountOfPlacedDocumentsPerPeer := map[yacymodel.Hash]int{}
	placedDocuments := make([]queryanswers.FoundDocument, 0, len(unplacedDocuments))
	for len(unplacedDocuments) > 0 {
		position := positionOfTheHighestDiscountedAgreementAmong(
			unplacedDocuments, agreementPerDocument, amountOfPlacedDocumentsPerPeer,
		)
		placedDocuments = append(placedDocuments, unplacedDocuments[position].document)
		for _, peer := range unplacedDocuments[position].peers {
			amountOfPlacedDocumentsPerPeer[peer]++
		}
		unplacedDocuments = slices.Delete(unplacedDocuments, position, position+1)
	}

	return placedDocuments
}

type documentListedByPeers struct {
	document queryanswers.FoundDocument
	peers    []yacymodel.Hash
}

func documentsListedByPeersOf(
	foundDocuments []queryanswers.FoundDocument,
) []documentListedByPeers {
	documentsListedByPeers := make([]documentListedByPeers, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		documentsListedByPeers = append(documentsListedByPeers, documentListedByPeers{
			document: foundDocument,
			peers:    peersThatListed(foundDocument),
		})
	}

	return documentsListedByPeers
}

func peersThatListed(foundDocument queryanswers.FoundDocument) []yacymodel.Hash {
	var peers []yacymodel.Hash
	for _, placeGivenByPeer := range foundDocument.PlacesGivenByPeers {
		if slices.Contains(peers, placeGivenByPeer.Peer) {
			continue
		}
		peers = append(peers, placeGivenByPeer.Peer)
	}

	return peers
}

func positionOfTheHighestDiscountedAgreementAmong(
	documentsListedByPeers []documentListedByPeers,
	agreementPerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedDocumentsPerPeer map[yacymodel.Hash]int,
) int {
	positionOfTheHighestDiscountedAgreement := 0
	highestDiscountedAgreement := math.Inf(-1)
	for position, documentListedByPeers := range documentsListedByPeers {
		discountedAgreement := discountedAgreementOf(
			documentListedByPeers, agreementPerDocument, amountOfPlacedDocumentsPerPeer,
		)
		if discountedAgreement > highestDiscountedAgreement {
			highestDiscountedAgreement = discountedAgreement
			positionOfTheHighestDiscountedAgreement = position
		}
	}

	return positionOfTheHighestDiscountedAgreement
}

func discountedAgreementOf(
	documentListedByPeers documentListedByPeers,
	agreementPerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedDocumentsPerPeer map[yacymodel.Hash]int,
) float64 {
	return agreementPerDocument[documentListedByPeers.document.Hash] * math.Pow(
		shareOfAgreementKeptPerPlacedDocumentOfTheSamePeer,
		float64(fewestPlacedDocumentsOf(
			documentListedByPeers.peers, amountOfPlacedDocumentsPerPeer,
		)),
	)
}

func fewestPlacedDocumentsOf(
	peers []yacymodel.Hash,
	amountOfPlacedDocumentsPerPeer map[yacymodel.Hash]int,
) int {
	if len(peers) == 0 {
		return 0
	}

	fewestPlacedDocuments := amountOfPlacedDocumentsPerPeer[peers[0]]
	for _, peer := range peers[1:] {
		fewestPlacedDocuments = min(fewestPlacedDocuments, amountOfPlacedDocumentsPerPeer[peer])
	}

	return fewestPlacedDocuments
}
