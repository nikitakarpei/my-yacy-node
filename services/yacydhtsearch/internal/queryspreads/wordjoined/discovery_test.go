package wordjoined_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const sixteenPartitionsOfTheRing = 16

func addressesInPartition(
	t *testing.T,
	partitions yacymodel.DHTRingPartitions,
	partition uint,
	amount int,
) []string {
	t.Helper()

	addresses := make([]string, 0, amount)
	for place := 0; len(addresses) < amount; place++ {
		address := fmt.Sprintf("https://document-%d.example/", place)
		if partitions.PartitionOf(documentHashOf(t, address)) != partition {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func settingsOfTwoPartitions() spreadSettings {
	settings := settingsOfOnePartition()
	settings.partitions = twoPartitionsOfTheRing
	settings.choice = responsiblePeers{
		peerAddressesPerWord: map[string][]string{
			firstWord:  {"first-in-0", "first-in-1"},
			secondWord: {"second-in-0", "second-in-1"},
		},
		partitionOfEachPeer: map[string]uint{
			"first-in-0":  0,
			"first-in-1":  1,
			"second-in-0": 0,
			"second-in-1": 1,
		},
	}

	return settings
}

func asksOfTheWord(
	spelledWord string,
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.SearchDocumentsAsk {
	var asksOfTheWord []peerasks.SearchDocumentsAsk
	for _, ask := range asks {
		if ask.Word != yacymodel.WordHash(spelledWord) {
			continue
		}
		asksOfTheWord = append(asksOfTheWord, ask)
	}

	return asksOfTheWord
}

func partitionsAskedAmong(asks []peerasks.SearchDocumentsAsk) []uint {
	partitionsAsked := make([]uint, 0, len(asks))
	for _, ask := range asks {
		partitionsAsked = append(partitionsAsked, ask.Partition)
	}

	return slices.Compact(slices.Sorted(slices.Values(partitionsAsked)))
}

func asksNamingDocumentsToMatchAmong(
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.SearchDocumentsAsk {
	var asksNamingDocuments []peerasks.SearchDocumentsAsk
	for _, ask := range asks {
		if len(ask.DocumentsToMatch) == 0 {
			continue
		}
		asksNamingDocuments = append(asksNamingDocuments, ask)
	}

	return asksNamingDocuments
}

func TestTheRarestSampledWordLeads(t *testing.T) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 3)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: documentsInPartitionZero},
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero[:1]},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})
	observer := &recordedSpreads{}

	settingsOfTwoPartitions().spread(network, observer)

	discoveryRound := observer.performed[0].DiscoveryRound
	if discoveryRound.LeadingQueryWordChoice != wordjoined.RarestSampledQueryWord ||
		discoveryRound.AmountOfSampledQueryWords != 2 ||
		discoveryRound.SampledPartition != 0 {
		t.Fatalf(
			"the spread reported %+v, want both words sampled in partition 0 and the rarest leading",
			discoveryRound,
		)
	}
	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.searchDocumentsAsks)
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Word != yacymodel.WordHash(firstWord) {
		t.Fatalf(
			"the spread put %v with documents to match, want one ask of the more common word",
			asksNamingDocuments,
		)
	}
}

func TestOnlyTheDocumentsOfTheSampledPartitionCountInASample(t *testing.T) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 2)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 4)
	network := networkOf(map[string]map[string][]string{
		"first-in-0": {
			firstWord: append(documentsInPartitionZero[:1:1], documentsInPartitionOne...),
		},
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})

	settingsOfTwoPartitions().spread(network, &recordedSpreads{})

	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.searchDocumentsAsks)
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Word != yacymodel.WordHash(secondWord) {
		t.Fatalf(
			"the spread put %v with documents to match, want the word with fewer documents "+
				"in the sampled partition to lead",
			asksNamingDocuments,
		)
	}
}

func TestAWordWithoutACompleteSampleCannotLead(t *testing.T) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 3)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: documentsInPartitionZero[:1]},
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})
	network.documentsPerAnswerOfEachPeer = map[string]int{"first-in-0": 0}
	observer := &recordedSpreads{}

	settingsOfTwoPartitions().spread(network, observer)

	if sampled := observer.performed[0].DiscoveryRound.AmountOfSampledQueryWords; sampled != 1 {
		t.Fatalf(
			"the spread sampled %d words, want only the word with a complete abstract",
			sampled,
		)
	}
	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.searchDocumentsAsks)
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Word != yacymodel.WordHash(firstWord) {
		t.Fatalf(
			"the spread put %v with documents to match, want the word without a sample asked "+
				"for the documents to match of the word with one",
			asksNamingDocuments,
		)
	}
}

func TestWithoutASampleEveryWordIsAskedWholeInEveryPartition(t *testing.T) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 2)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: documentsInPartitionZero[:1]},
		"first-in-1":  {firstWord: documentsInPartitionOne[:2]},
		"second-in-0": {secondWord: documentsInPartitionZero},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})
	network.peersCountingNoDocument = map[string]struct{}{"first-in-0": {}, "second-in-0": {}}
	observer := &recordedSpreads{}

	answeredQuery := settingsOfTwoPartitions().spread(network, observer)

	if asksNamingDocuments := asksNamingDocumentsToMatchAmong(
		network.searchDocumentsAsks,
	); len(asksNamingDocuments) != 0 || len(network.searchDocumentsAsks) != 4 {
		t.Fatalf(
			"the spread put %v, want every word asked whole in every partition",
			network.searchDocumentsAsks,
		)
	}
	discoveryRound := observer.performed[0].DiscoveryRound
	if discoveryRound.LeadingQueryWordChoice != wordjoined.RarestQueryWordWithoutASample ||
		len(discoveryRound.OtherWordAsksPerPartition) != 0 {
		t.Fatalf(
			"the spread reported %+v, want the rarest counted word leading without a sample "+
				"and no other word asks per partition",
			discoveryRound,
		)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(
		append(documentsInPartitionZero[:1:1], documentsInPartitionOne[:2]...),
	))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents both words hold %v", got, wanted)
	}
}

func networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(
	t *testing.T,
) (*peerNetwork, []string) {
	t.Helper()

	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: {}},
		"first-in-1":  {firstWord: documentsInPartitionOne[:2]},
		"second-in-0": {secondWord: addressesInPartition(t, twoPartitionsOfTheRing, 0, 1)},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})

	return network, documentsInPartitionOne[:2]
}

func TestTheOtherWordsAreAskedOnlyInPartitionsWithDocumentsToMatchAndForThem(t *testing.T) {
	t.Parallel()

	network, documentsToMatch := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	observer := &recordedSpreads{}

	answeredQuery := settingsOfTwoPartitions().spread(network, observer)

	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.searchDocumentsAsks)
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsToMatch))
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Peer.Address != "second-in-1" ||
		!slices.Equal(documentsInTheirHashOrder(asksNamingDocuments[0].DocumentsToMatch), wanted) {
		t.Fatalf(
			"the spread put %v with documents to match, want one ask to the replica in "+
				"partition 1 naming the documents to match %v",
			asksNamingDocuments,
			wanted,
		)
	}
	wantedAsks := map[uint]wordjoined.OtherWordAsks{
		1: wordjoined.OtherWordAsksNamingTheDocumentsToMatch,
	}
	if got := observer.performed[0].DiscoveryRound.OtherWordAsksPerPartition; !maps.Equal(
		got, wantedAsks,
	) {
		t.Fatalf("the spread reported %v, want %v", got, wantedAsks)
	}
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents to match both words hold %v", got, wanted)
	}
}

func TestAPartitionWithMoreDocumentsToMatchThanTheCeilingIsAskedWhole(t *testing.T) {
	t.Parallel()

	network, documentsToMatch := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	observer := &recordedSpreads{}
	settings := settingsOfTwoPartitions()
	settings.documentsToMatchCeiling = 1

	answeredQuery := settings.spread(network, observer)

	if asksNamingDocuments := asksNamingDocumentsToMatchAmong(
		network.searchDocumentsAsks,
	); len(asksNamingDocuments) != 0 {
		t.Fatalf("the spread put %v, want no ask naming documents to match", asksNamingDocuments)
	}
	if got := observer.performed[0].DiscoveryRound.OtherWordAsksPerPartition[1]; got !=
		wordjoined.OtherWordAsksOverTheCeiling {
		t.Fatalf("the spread reported partition 1 as %q, want it over the ceiling", got)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsToMatch))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents to match both words hold %v", got, wanted)
	}
}

func TestAPartitionWhereTheLeadingWordIsPartialIsAskedForTheDocumentsItListed(t *testing.T) {
	t.Parallel()

	network, documentsToMatch := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	network.documentsPerAnswerOfEachPeer = map[string]int{"first-in-1": 1}
	observer := &recordedSpreads{}

	settingsOfTwoPartitions().spread(network, observer)

	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.searchDocumentsAsks)
	if len(asksNamingDocuments) != 1 ||
		!slices.Equal(
			asksNamingDocuments[0].DocumentsToMatch,
			documentHashesOf(documentsToMatch[:1]),
		) {
		t.Fatalf(
			"the spread put %v, want one ask naming the listed document %v",
			asksNamingDocuments, documentsToMatch[:1],
		)
	}
	if got := observer.performed[0].DiscoveryRound.OtherWordAsksPerPartition[1]; got !=
		wordjoined.OtherWordAsksNamingTheDocumentsToMatch {
		t.Fatalf(
			"the spread reported partition 1 as %q, want its other word asks naming the documents to match",
			got,
		)
	}
}

func TestAnAnswerThatIgnoresTheDocumentsToMatchJoinsOnlyTheDocumentsEveryWordHolds(
	t *testing.T,
) {
	t.Parallel()

	network, documentsToMatch := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	network.peersListingDocumentsTheAskDidNotName = map[string]struct{}{"second-in-1": {}}
	observer := &recordedSpreads{}

	answeredQuery := settingsOfTwoPartitions().spread(network, observer)

	wanted := documentsInTheirHashOrder(documentHashesOf(documentsToMatch))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf(
			"the spread found %v, want only the documents to match both words hold %v",
			got,
			wanted,
		)
	}
}

func TestAJoinedDocumentAnAskForTheDocumentsToMatchAnsweredIsNotAskedMetadataFor(t *testing.T) {
	t.Parallel()

	network, documentsToMatch := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	network.answeredItemsPerWordPerPeer = map[string]map[string][]string{
		"second-in-1": {secondWord: documentsToMatch[:1]},
	}

	settingsOfTwoPartitions().spread(network, &recordedSpreads{})

	wanted := documentHashesOf(documentsToMatch[1:])
	if got := distinctDocumentsAskedMetadataFor(network.urlMetadataAsks); !slices.Equal(
		got, wanted,
	) {
		t.Fatalf(
			"the spread asked metadata for %v, want only the document to match no answer carried",
			got,
		)
	}
}

func TestACompoundWordIsAskedInEveryPartitionOnlyWhenItHoldsTheLeadingWord(t *testing.T) {
	t.Parallel()

	documentInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 1)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 2)
	network := networkOf(map[string]map[string][]string{
		"in-0": {secondWord: documentInPartitionZero, thirdWord: documentInPartitionZero},
		"in-1": {
			firstWord:  documentsInPartitionOne,
			secondWord: documentsInPartitionOne,
			thirdWord:  documentsInPartitionOne,
		},
	})
	settings := settingsOfOnePartition()
	settings.query = firstWord + " " + secondWord + " " + thirdWord
	settings.partitions = twoPartitionsOfTheRing
	settings.askablePeers = []string{"in-0", "in-1"}
	settings.choice = responsiblePeers{partitionOfEachPeer: map[string]uint{"in-0": 0, "in-1": 1}}

	answeredQuery := settings.spread(network, &recordedSpreads{})

	asksOfTheCompoundWithTheLead := asksOfTheWord(firstWord+secondWord, network.searchDocumentsAsks)
	if got := partitionsAskedAmong(asksOfTheCompoundWithTheLead); !slices.Equal(
		got, []uint{0, 1},
	) || len(asksNamingDocumentsToMatchAmong(asksOfTheCompoundWithTheLead)) != 0 {
		t.Fatalf(
			"the spread put %v, want the compound word of the leading word asked whole in "+
				"every partition",
			asksOfTheCompoundWithTheLead,
		)
	}
	asksOfTheOtherCompound := asksOfTheWord(secondWord+thirdWord, network.searchDocumentsAsks)
	if got := partitionsAskedAmong(asksOfTheOtherCompound); !slices.Equal(got, []uint{1}) ||
		len(asksNamingDocumentsToMatchAmong(asksOfTheOtherCompound)) != 1 {
		t.Fatalf(
			"the spread put %v, want the other compound word asked for the documents to match of "+
				"partition 1 only",
			asksOfTheOtherCompound,
		)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsInPartitionOne))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents every word holds %v", got, wanted)
	}
}

func networkOfSixteenPartitions(t *testing.T) (*peerNetwork, []string, spreadSettings) {
	t.Helper()

	documentsPerWordPerPeer := map[string]map[string][]string{}
	partitionOfEachPeer := map[string]uint{}
	var addresses, documentsEveryWordHolds []string
	for partition := range uint(sixteenPartitionsOfTheRing) {
		address := fmt.Sprintf("in-%d", partition)
		addresses = append(addresses, address)
		partitionOfEachPeer[address] = partition
		documents := addressesInPartition(t, sixteenPartitionsOfTheRing, partition, 3)
		documentsPerWordPerPeer[address] = map[string][]string{secondWord: documents}
		if partition%2 == 1 {
			continue
		}
		documentsPerWordPerPeer[address][firstWord] = documents[:2]
		documentsEveryWordHolds = append(documentsEveryWordHolds, documents[:2]...)
	}
	settings := settingsOfOnePartition()
	settings.partitions = sixteenPartitionsOfTheRing
	settings.sampledPartition = 4
	settings.askablePeers = addresses
	settings.choice = responsiblePeers{partitionOfEachPeer: partitionOfEachPeer}

	return networkOf(documentsPerWordPerPeer), documentsEveryWordHolds, settings
}

func TestTheJoinHoldsTheDocumentsEveryWordHoldsWhenTheLeadingWordIsCompleteEverywhere(
	t *testing.T,
) {
	t.Parallel()

	network, documentsEveryWordHolds, settings := networkOfSixteenPartitions(t)

	answeredQuery := settings.spread(network, &recordedSpreads{})

	wanted := documentsInTheirHashOrder(documentHashesOf(documentsEveryWordHolds))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents every word holds %v", got, wanted)
	}
}

func TestTheSpreadAsksEveryWordPartitionAtMostOnce(t *testing.T) {
	t.Parallel()

	network, _, settings := networkOfSixteenPartitions(t)
	const amountOfWordsAndCompoundWords = 3

	settings.spread(network, &recordedSpreads{})

	if amountOfAsks := len(network.searchDocumentsAsks); amountOfAsks >
		sixteenPartitionsOfTheRing*amountOfWordsAndCompoundWords {
		t.Fatalf(
			"the spread put %d asks, want at most one for each partition of each word",
			amountOfAsks,
		)
	}
	if partitionsOfTheOtherWord := partitionsAskedAmong(
		asksOfTheWord(secondWord, network.searchDocumentsAsks),
	); len(partitionsOfTheOtherWord) != sixteenPartitionsOfTheRing/2 {
		t.Fatalf(
			"the spread asked the other word in the partitions %v, want only the partitions "+
				"with documents to match",
			partitionsOfTheOtherWord,
		)
	}
}
