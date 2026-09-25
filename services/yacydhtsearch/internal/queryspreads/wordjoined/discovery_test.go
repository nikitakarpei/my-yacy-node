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
	asks []peerasks.WordAbstractAsk,
) []peerasks.WordAbstractAsk {
	var asksOfTheWord []peerasks.WordAbstractAsk
	for _, ask := range asks {
		if ask.Word != yacymodel.WordHash(spelledWord) {
			continue
		}
		asksOfTheWord = append(asksOfTheWord, ask)
	}

	return asksOfTheWord
}

func partitionsAskedAmong(asks []peerasks.WordAbstractAsk) []uint {
	partitionsAsked := make([]uint, 0, len(asks))
	for _, ask := range asks {
		partitionsAsked = append(partitionsAsked, ask.Partition)
	}

	return slices.Compact(slices.Sorted(slices.Values(partitionsAsked)))
}

func asksNamingDocumentsToMatchAmong(
	asks []peerasks.WordAbstractAsk,
) []peerasks.WordAbstractAsk {
	var asksNamingDocuments []peerasks.WordAbstractAsk
	for _, ask := range asks {
		if len(ask.DocumentsToMatch) == 0 {
			continue
		}
		asksNamingDocuments = append(asksNamingDocuments, ask)
	}

	return asksNamingDocuments
}

func TestTheRarestWordWithASampleLeads(t *testing.T) {
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
	if discoveryRound.LeadingQueryWordChoice != wordjoined.RarestQueryWordWithASample ||
		discoveryRound.AmountOfQueryWordsWithASample != 2 ||
		discoveryRound.SampledPartition != 0 {
		t.Fatalf(
			"the spread reported %+v, want both words with a sample in partition 0 and the rarest leading",
			discoveryRound,
		)
	}
	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.wordAbstractAsks)
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

	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.wordAbstractAsks)
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Word != yacymodel.WordHash(secondWord) {
		t.Fatalf(
			"the spread put %v with documents to match, want the word with fewer documents "+
				"in the sampled partition to lead",
			asksNamingDocuments,
		)
	}
}

func TestAWordWithoutASampleCannotLead(t *testing.T) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 3)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-0":  {firstWord: documentsInPartitionZero[:1]},
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})
	network.silentPeers["first-in-0"] = struct{}{}
	observer := &recordedSpreads{}

	settingsOfTwoPartitions().spread(network, observer)

	if amountOfQueryWordsWithASample := observer.performed[0].DiscoveryRound.
		AmountOfQueryWordsWithASample; amountOfQueryWordsWithASample != 1 {
		t.Fatalf(
			"the spread reported %d words with a sample, want only the word whose replica "+
				"answered in the sampled partition",
			amountOfQueryWordsWithASample,
		)
	}
	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.wordAbstractAsks)
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Word != yacymodel.WordHash(firstWord) {
		t.Fatalf(
			"the spread put %v with documents to match, want the word without a sample asked "+
				"for the documents to match of the word with one",
			asksNamingDocuments,
		)
	}
}

func TestAWordAReplicaHoldsNothingForIsASampleOfNoDocuments(t *testing.T) {
	t.Parallel()

	documentsInPartitionZero := addressesInPartition(t, twoPartitionsOfTheRing, 0, 3)
	documentsInPartitionOne := addressesInPartition(t, twoPartitionsOfTheRing, 1, 3)
	network := networkOf(map[string]map[string][]string{
		"first-in-1":  {firstWord: documentsInPartitionOne},
		"second-in-0": {secondWord: documentsInPartitionZero},
		"second-in-1": {secondWord: documentsInPartitionOne},
	})
	observer := &recordedSpreads{}

	settingsOfTwoPartitions().spread(network, observer)

	discoveryRound := observer.performed[0].DiscoveryRound
	if discoveryRound.AmountOfQueryWordsWithASample != 2 {
		t.Fatalf("the spread reported %+v, want both words with a sample", discoveryRound)
	}
	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.wordAbstractAsks)
	if len(asksNamingDocuments) != 1 ||
		asksNamingDocuments[0].Word != yacymodel.WordHash(secondWord) {
		t.Fatalf(
			"the spread put %v with documents to match, want the word with no documents leading",
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
	network.silentPeers = map[string]struct{}{"first-in-0": {}, "second-in-0": {}}
	observer := &recordedSpreads{}

	answeredQuery := settingsOfTwoPartitions().spread(network, observer)

	if asksNamingDocuments := asksNamingDocumentsToMatchAmong(
		network.wordAbstractAsks,
	); len(asksNamingDocuments) != 0 || len(network.wordAbstractAsks) != 4 {
		t.Fatalf(
			"the spread put %v, want every word asked whole in every partition",
			network.wordAbstractAsks,
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
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsInPartitionOne[:2]))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents both answered words hold %v", got, wanted)
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

	asksNamingDocuments := asksNamingDocumentsToMatchAmong(network.wordAbstractAsks)
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
		network.wordAbstractAsks,
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

	asksOfTheCompoundWithTheLead := asksOfTheWord(firstWord+secondWord, network.wordAbstractAsks)
	if got := partitionsAskedAmong(asksOfTheCompoundWithTheLead); !slices.Equal(
		got, []uint{0, 1},
	) || len(asksNamingDocumentsToMatchAmong(asksOfTheCompoundWithTheLead)) != 0 {
		t.Fatalf(
			"the spread put %v, want the compound word of the leading word asked whole in "+
				"every partition",
			asksOfTheCompoundWithTheLead,
		)
	}
	asksOfTheOtherCompound := asksOfTheWord(secondWord+thirdWord, network.wordAbstractAsks)
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

	if amountOfAsks := len(network.wordAbstractAsks); amountOfAsks >
		sixteenPartitionsOfTheRing*amountOfWordsAndCompoundWords {
		t.Fatalf(
			"the spread put %d asks, want at most one for each partition of each word",
			amountOfAsks,
		)
	}
	if partitionsOfTheOtherWord := partitionsAskedAmong(
		asksOfTheWord(secondWord, network.wordAbstractAsks),
	); len(partitionsOfTheOtherWord) != sixteenPartitionsOfTheRing/2 {
		t.Fatalf(
			"the spread asked the other word in the partitions %v, want only the partitions "+
				"with documents to match",
			partitionsOfTheOtherWord,
		)
	}
}

func settledWordPartitionsReadAtTheAsksOf(spelledWord string, network *peerNetwork) []int {
	var settledWordPartitionsRead []int
	for place, ask := range network.wordAbstractAsks {
		if ask.Word != yacymodel.WordHash(spelledWord) {
			continue
		}
		settledWordPartitionsRead = append(
			settledWordPartitionsRead, network.settledWordPartitionsReadAtEachAsk[place],
		)
	}

	return settledWordPartitionsRead
}

func settingsWithTheLeadingWordRemembered(rememberedAmountOfTheLeadingWord int) spreadSettings {
	settings := settingsOfTwoPartitions()
	settings.documentsToMatchCeiling = 1
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{
		firstWord: rememberedAmountOfTheLeadingWord, secondWord: 100,
	})

	return settings
}

func TestARememberedLeadingWordOverTheCeilingInEveryPartitionHasTheOtherWordsAskedAtTheStart(
	t *testing.T,
) {
	t.Parallel()

	network, documentsToMatch := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	observer := &recordedSpreads{}

	answeredQuery := settingsWithTheLeadingWordRemembered(4).spread(network, observer)

	asksOfTheOtherWord := asksOfTheWord(secondWord, network.wordAbstractAsks)
	if got := settledWordPartitionsReadAtTheAsksOf(secondWord, network); !slices.Equal(
		got, []int{0, 0},
	) || len(asksNamingDocumentsToMatchAmong(asksOfTheOtherWord)) != 0 {
		t.Fatalf(
			"the spread put %v after %v settled word partitions, want the other word asked "+
				"whole in both partitions before any partition settled",
			asksOfTheOtherWord, got,
		)
	}
	wantedAsks := map[uint]wordjoined.OtherWordAsks{
		0: wordjoined.OtherWordAsksPredictedOverTheCeiling,
		1: wordjoined.OtherWordAsksPredictedOverTheCeiling,
	}
	if got := observer.performed[0].DiscoveryRound.OtherWordAsksPerPartition; !maps.Equal(
		got, wantedAsks,
	) {
		t.Fatalf("the spread reported %v, want %v", got, wantedAsks)
	}
	wanted := documentsInTheirHashOrder(documentHashesOf(documentsToMatch))
	if got := foundDocumentsIn(answeredQuery); !slices.Equal(got, wanted) {
		t.Fatalf("the spread found %v, want the documents to match both words hold %v", got, wanted)
	}
}

func TestARememberedLeadingWordAtTheCeilingInEveryPartitionHasTheOtherWordsWaitForThePartition(
	t *testing.T,
) {
	t.Parallel()

	network, _ := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)

	settingsWithTheLeadingWordRemembered(2).spread(network, &recordedSpreads{})

	if got := settledWordPartitionsReadAtTheAsksOf(secondWord, network); len(got) != 1 ||
		got[0] == 0 {
		t.Fatalf(
			"the spread asked the other word after %v settled word partitions, want one ask "+
				"after its partition settled",
			got,
		)
	}
}

func TestASampledLeadingWordHasTheOtherWordsWaitForThePartitionWhateverItsRememberedAmount(
	t *testing.T,
) {
	t.Parallel()

	network, _ := networkWhereTheLeadingWordHoldsDocumentsOnlyInPartitionOne(t)
	settings := settingsOfTwoPartitions()
	settings.documentsToMatchCeiling = 1
	settings.queryWordDocumentAmounts = queryWordDocumentAmountsOf(map[string]int{firstWord: 100})
	observer := &recordedSpreads{}

	settings.spread(network, observer)

	wantedAsks := map[uint]wordjoined.OtherWordAsks{1: wordjoined.OtherWordAsksOverTheCeiling}
	if got := observer.performed[0].DiscoveryRound.OtherWordAsksPerPartition; !maps.Equal(
		got, wantedAsks,
	) {
		t.Fatalf("the spread reported %v, want %v", got, wantedAsks)
	}
}
