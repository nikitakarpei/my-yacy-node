package peerchoice_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	partitionExponent = 4
	ringPartitions    = 1 << partitionExponent
	networkRedundancy = 2
)

type reliabilityPerPeer map[yacymodel.Hash]float64

func (reliability reliabilityPerPeer) ReliabilityOf(
	_ context.Context,
	peerAtAddress probeanswerhistory.PeerAtAddress,
) float64 {
	return reliability[peerAtAddress.Hash]
}

type recordedFractions struct{ fractions [][]float64 }

func (recorded *recordedFractions) PeersTakenFromTheRing(_ context.Context, fractions []float64) {
	recorded.fractions = append(recorded.fractions, fractions)
}

func partitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	count, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("partitions from exponent: %v", err)
	}

	return count
}

func choiceOver(
	t *testing.T,
	networkRedundancy int,
	reliability reliabilityPerPeer,
	observer peerchoice.PeerChoiceObserver,
) peerchoice.Choice {
	t.Helper()

	return peerchoice.New(partitions(t), networkRedundancy, reliability, observer)
}

func askablePeers(t *testing.T, count int) []peerdirectory.AskablePeer {
	t.Helper()

	peers := make([]peerdirectory.AskablePeer, 0, count)
	for index := range count {
		peers = append(peers, peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash("peer-" + strconv.Itoa(index)),
			Address: "http://10.0.0.1:8090",
		})
	}

	return peers
}

func words(t *testing.T, spelled ...string) []yacymodel.Hash {
	t.Helper()

	hashes := make([]yacymodel.Hash, 0, len(spelled))
	for _, word := range spelled {
		hashes = append(hashes, yacymodel.WordHash(word))
	}

	return hashes
}

func peersOf(chosenPeers []peerchoice.ChosenPeer) []peerdirectory.AskablePeer {
	peers := make([]peerdirectory.AskablePeer, 0, len(chosenPeers))
	for _, chosenPeer := range chosenPeers {
		peers = append(peers, chosenPeer.Peer)
	}

	return peers
}

func TestEveryChosenPeerIsNamedOnceForItsQueryWord(t *testing.T) {
	t.Parallel()

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(
		t.Context(), words(t, "berlin"), askablePeers(t, 20),
	)

	seen := map[yacymodel.Hash]struct{}{}
	for _, peer := range peersOf(peersPerQueryWord[0].ChosenPeers) {
		if _, twice := seen[peer.Hash]; twice {
			t.Fatalf("the query word was given %s twice", peer.Hash)
		}
		seen[peer.Hash] = struct{}{}
	}
}

func TestNoMorePeersAreChosenForAQueryWordThanTheCeilingAllows(t *testing.T) {
	t.Parallel()

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(
		t.Context(), words(t, "berlin"), askablePeers(t, 200),
	)

	if len(peersOf(peersPerQueryWord[0].ChosenPeers)) != networkRedundancy*ringPartitions {
		t.Fatalf(
			"the query word was given %d peers, want %d",
			len(peersOf(peersPerQueryWord[0].ChosenPeers)),
			networkRedundancy*ringPartitions,
		)
	}
}

func TestEveryPartitionOfTheRingGivesAPeerBeforeAnyGivesASecond(t *testing.T) {
	t.Parallel()

	observer := &recordedFractions{}

	choiceOver(
		t,
		networkRedundancy,
		reliabilityPerPeer{},
		observer,
	).ChosenPeersPerQueryWordFor(
		t.Context(), words(t, "berlin"), askablePeers(t, 200),
	)

	for index, fraction := range observer.fractions[0][:ringPartitions] {
		if fraction >= 1.0/ringPartitions {
			t.Fatalf(
				"peer %d of the first turn sits %v of the ring from the word, want inside its own partition",
				index,
				fraction,
			)
		}
	}
}

func TestEveryAskablePeerIsChosenWhenTheyAreFewerThanTheCeiling(t *testing.T) {
	t.Parallel()

	askable := askablePeers(t, 2)

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), words(t, "berlin"), askable)

	if len(peersOf(peersPerQueryWord[0].ChosenPeers)) != len(askable) {
		t.Fatalf(
			"the query word was given %d of %d askable peers, want all",
			len(peersOf(peersPerQueryWord[0].ChosenPeers)),
			len(askable),
		)
	}
}

func TestNoPeerIsChosenFromAnEmptyAskableSet(t *testing.T) {
	t.Parallel()

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), words(t, "berlin"), nil)

	if len(peersOf(peersPerQueryWord[0].ChosenPeers)) != 0 {
		t.Fatalf(
			"the query word was given %v, want no peer",
			peersOf(peersPerQueryWord[0].ChosenPeers),
		)
	}
}

func TestOneRingFractionIsReportedForEachPeerTheRingChose(t *testing.T) {
	t.Parallel()

	observer := &recordedFractions{}

	peersPerQueryWord := choiceOver(
		t, networkRedundancy,
		reliabilityPerPeer{},
		peerchoice.PeerChoiceObservers{observer},
	).ChosenPeersPerQueryWordFor(
		t.Context(), words(t, "berlin"), askablePeers(t, 20),
	)

	if len(observer.fractions[0]) != len(peersOf(peersPerQueryWord[0].ChosenPeers)) {
		t.Fatalf(
			"PeersTakenFromTheRing reported %d fractions for %d peers",
			len(observer.fractions[0]),
			len(peersOf(peersPerQueryWord[0].ChosenPeers)),
		)
	}
	for _, fraction := range observer.fractions[0] {
		if fraction < 0 || fraction > 1 {
			t.Fatalf("ring fraction %v is outside the ring", fraction)
		}
	}
}

func TestAReliablePeerNeverTakesTheSlotOfAPeerNearerToTheWord(t *testing.T) {
	t.Parallel()

	askable := askablePeers(t, 200)
	word := words(t, "berlin")
	knownToNobody := choiceOver(
		t, 1, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), word, askable)

	chosen := choiceOver(
		t, 1, reliableOutside(askable, peersOf(knownToNobody[0].ChosenPeers)),
		&recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), word, askable)

	if somePeerIsOutside(peersOf(chosen[0].ChosenPeers), peersOf(knownToNobody[0].ChosenPeers)) {
		t.Fatalf(
			"the query chose %v, want only the nearest peers of each partition %v",
			peersOf(chosen[0].ChosenPeers),
			peersOf(knownToNobody[0].ChosenPeers),
		)
	}
}

func TestAReliablePeerIsAskedBeforeALessReliablePeerOfItsPartition(t *testing.T) {
	t.Parallel()

	askable := askablePeers(t, 200)
	word := words(t, "berlin")
	knownToNobody := choiceOver(
		t, 2, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), word, askable)
	secondTurn := peersOf(knownToNobody[0].ChosenPeers)[ringPartitions:]

	chosen := choiceOver(
		t,
		2,
		reliableOutside(askable, peersOf(knownToNobody[0].ChosenPeers)[:ringPartitions]),
		&recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), word, askable)

	if somePeerIsOutside(peersOf(chosen[0].ChosenPeers)[:ringPartitions], secondTurn) {
		t.Fatalf(
			"the first turn asked %v, want the reliable peers %v first",
			peersOf(chosen[0].ChosenPeers)[:ringPartitions],
			secondTurn,
		)
	}
}

func reliableOutside(
	askable []peerdirectory.AskablePeer,
	unreliable []peerdirectory.AskablePeer,
) reliabilityPerPeer {
	reliable := reliabilityPerPeer{}
	for _, peer := range askable {
		if somePeerIsOutside([]peerdirectory.AskablePeer{peer}, unreliable) {
			reliable[peer.Hash] = 1
		}
	}

	return reliable
}

func TestTwoSearchesForOneWordAskTheSamePeers(t *testing.T) {
	t.Parallel()

	askable := askablePeers(t, 200)
	word := words(t, "berlin")
	firstSearch := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), word, askable)
	secondSearch := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), word, askable)

	if somePeerIsOutside(
		peersOf(secondSearch[0].ChosenPeers),
		peersOf(firstSearch[0].ChosenPeers),
	) {
		t.Fatalf(
			"the searches asked %v and %v, want the same peers",
			peersOf(firstSearch[0].ChosenPeers),
			peersOf(secondSearch[0].ChosenPeers),
		)
	}
}

func somePeerIsOutside(
	peers []peerdirectory.AskablePeer,
	chosen []peerdirectory.AskablePeer,
) bool {
	for _, peer := range peers {
		inside := false
		for _, chosenPeer := range chosen {
			inside = inside || chosenPeer.Hash == peer.Hash
		}
		if !inside {
			return true
		}
	}

	return false
}

func TestALaterQueryWordReachesPeersTheEarlierWordsDidNot(t *testing.T) {
	t.Parallel()

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(
		t.Context(), words(t, "berlin", "berlin"), askablePeers(t, 200),
	)

	for _, peer := range peersOf(peersPerQueryWord[1].ChosenPeers) {
		if !somePeerIsOutside(
			[]peerdirectory.AskablePeer{peer},
			peersOf(peersPerQueryWord[0].ChosenPeers),
		) {
			t.Fatalf(
				"%s answers the second word although the first word already asked it",
				peer.Hash,
			)
		}
	}
}

func TestALaterQueryWordTakesPeersAnEarlierWordChoseWhenNoOtherPeerIsLeft(t *testing.T) {
	t.Parallel()

	askable := askablePeers(t, 2)

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(t.Context(), words(t, "berlin", "weather"), askable)

	if len(peersOf(peersPerQueryWord[1].ChosenPeers)) != len(askable) {
		t.Fatalf(
			"the second word was given %d of %d askable peers, want all",
			len(peersOf(peersPerQueryWord[1].ChosenPeers)),
			len(askable),
		)
	}
}

func TestThePeersOfAQueryWordSpanEveryPartitionOfTheRing(t *testing.T) {
	t.Parallel()

	peersPerQueryWord := choiceOver(
		t, networkRedundancy, reliabilityPerPeer{}, &recordedFractions{},
	).ChosenPeersPerQueryWordFor(
		t.Context(), words(t, "berlin"), askablePeers(t, 200),
	)

	partitionsWithAPeer := map[uint]struct{}{}
	for _, chosenPeer := range peersPerQueryWord[0].ChosenPeers {
		if chosenPeer.Partition >= ringPartitions {
			t.Fatalf(
				"%s sits in partition %d, want one of the %d of the ring",
				chosenPeer.Peer.Hash,
				chosenPeer.Partition,
				ringPartitions,
			)
		}
		partitionsWithAPeer[chosenPeer.Partition] = struct{}{}
	}
	if len(partitionsWithAPeer) != ringPartitions {
		t.Fatalf(
			"the chosen peers sit in %d partitions of the ring, want all %d",
			len(partitionsWithAPeer),
			ringPartitions,
		)
	}
}

func TestAPeerChosenForTwoQueryWordsKeepsThePartitionOfItsFirstQueryWord(t *testing.T) {
	t.Parallel()

	chosenPeersPerQueryWord := peerchoice.ChosenPeersPerQueryWord{
		{
			QueryWord: yacymodel.WordHash("berlin"),
			ChosenPeers: []peerchoice.ChosenPeer{
				{Peer: askablePeers(t, 2)[0], Partition: 3},
				{Peer: askablePeers(t, 2)[1], Partition: 7},
			},
		},
		{
			QueryWord: yacymodel.WordHash("weather"),
			ChosenPeers: []peerchoice.ChosenPeer{
				{Peer: askablePeers(t, 2)[1], Partition: 11},
			},
		},
	}

	chosenPeersAcrossQueryWords := chosenPeersPerQueryWord.ChosenPeersAcrossQueryWords()

	want := []peerchoice.ChosenPeer{
		{Peer: askablePeers(t, 2)[0], Partition: 3},
		{Peer: askablePeers(t, 2)[1], Partition: 7},
	}
	if !slices.Equal(chosenPeersAcrossQueryWords, want) {
		t.Fatalf("the query words chose %v, want %v", chosenPeersAcrossQueryWords, want)
	}
}

func TestThePeersChosenForOneQueryWordLeaveOutThePeersOfTheOthers(t *testing.T) {
	t.Parallel()

	chosenPeersPerQueryWord := peerchoice.ChosenPeersPerQueryWord{
		{
			QueryWord:   yacymodel.WordHash("berlin"),
			ChosenPeers: []peerchoice.ChosenPeer{{Peer: askablePeers(t, 2)[0], Partition: 3}},
		},
		{
			QueryWord:   yacymodel.WordHash("weather"),
			ChosenPeers: []peerchoice.ChosenPeer{{Peer: askablePeers(t, 2)[1], Partition: 11}},
		},
	}

	chosenPeers := chosenPeersPerQueryWord.ChosenPeersOf(yacymodel.WordHash("weather"))

	want := []peerchoice.ChosenPeer{{Peer: askablePeers(t, 2)[1], Partition: 11}}
	if !slices.Equal(chosenPeers, want) {
		t.Fatalf("the query word chose %v, want %v", chosenPeers, want)
	}
}
