package matchingwords_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	twoPartitionsOfTheRing  = 2
	documentsToMatchCeiling = 2
	everyPartition          = -1
)

var (
	leadingWord  = yacymodel.WordHash("berlin")
	matchingWord = yacymodel.WordHash("weather")
	roles        = wordroles.Roles{
		ListingWords:  []yacymodel.Hash{leadingWord},
		MatchingWords: []yacymodel.Hash{matchingWord},
	}
)

type askOfTheRun struct {
	partition        int
	words            []yacymodel.Hash
	documentsToMatch []yacymodel.URLHash
	waitsBefore      int
}

type runOfTheReplicas struct {
	waitsUntilEachPartitionSettles map[uint]int
	documentsListedInEachPartition map[uint][]yacymodel.URLHash
	askedPartitions                map[uint]struct{}
	holders                        documentholders.Holders
	waits                          int
	asks                           []askOfTheRun
}

func runWhere(
	waitsUntilEachPartitionSettles map[uint]int,
	documentsListedInEachPartition map[uint][]yacymodel.URLHash,
) *runOfTheReplicas {
	return &runOfTheReplicas{
		waitsUntilEachPartitionSettles: waitsUntilEachPartitionSettles,
		documentsListedInEachPartition: documentsListedInEachPartition,
		askedPartitions:                map[uint]struct{}{},
		holders:                        documentholders.Holders{},
	}
}

func (run *runOfTheReplicas) PartitionSettledFor(partition uint, _ []yacymodel.Hash) bool {
	return run.waits >= run.waitsUntilEachPartitionSettles[partition]
}

func (run *runOfTheReplicas) PartitionAskedFor(partition uint, _ []yacymodel.Hash) bool {
	_, asked := run.askedPartitions[partition]

	return asked
}

func (run *runOfTheReplicas) WaitUntilAnyPartitionSettles() {
	run.waits++
}

func (run *runOfTheReplicas) DocumentsListedIn(
	partition uint,
	_ []yacymodel.Hash,
) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	documents.AddEach(run.documentsListedInEachPartition[partition])

	return documents
}

func (run *runOfTheReplicas) DocumentHolders() documentholders.Holders {
	return run.holders
}

func (run *runOfTheReplicas) AskEveryPartitionFor(words []yacymodel.Hash) {
	for partition := range uint(twoPartitionsOfTheRing) {
		run.askedPartitions[partition] = struct{}{}
	}
	run.asks = append(run.asks, askOfTheRun{
		partition: everyPartition, words: words, waitsBefore: run.waits,
	})
}

func (run *runOfTheReplicas) AskPartitionFor(partition uint, words []yacymodel.Hash) {
	run.AskPartitionForWordsAmong(partition, words, nil)
}

func (run *runOfTheReplicas) AskPartitionForWordsAmong(
	partition uint,
	words []yacymodel.Hash,
	documentsToMatch []yacymodel.URLHash,
) {
	run.askedPartitions[partition] = struct{}{}
	run.asks = append(run.asks, askOfTheRun{
		partition:        int(partition),
		words:            words,
		documentsToMatch: documentsToMatch,
		waitsBefore:      run.waits,
	})
}

func (run *runOfTheReplicas) holdBy(peerAddress string, documents ...yacymodel.URLHash) {
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
	run.holders.AddHoldersIn([]wordpartitionasks.ReplicaAnswer{answer})
}

func (run *runOfTheReplicas) waitsBeforeEachAsk() []int {
	waitsBeforeEachAsk := make([]int, 0, len(run.asks))
	for _, ask := range run.asks {
		waitsBeforeEachAsk = append(waitsBeforeEachAsk, ask.waitsBefore)
	}

	return waitsBeforeEachAsk
}

func documentsOf(t *testing.T, amount int) []yacymodel.URLHash {
	t.Helper()

	documents := make([]yacymodel.URLHash, 0, amount)
	for place := range amount {
		document, err := yacymodel.URLHashOf(fmt.Sprintf("https://document-%d.example/", place))
		if err != nil {
			t.Fatalf("document %d has no hash: %v", place, err)
		}
		documents = append(documents, document)
	}

	return documents
}

func asker() matchingwords.Asker {
	return matchingwords.New(twoPartitionsOfTheRing, documentsToMatchCeiling)
}

func leadingWordWithoutAnAmount() leadingword.Lead {
	return leadingword.Lead{
		Word:                          yacymodel.Some(leadingWord),
		AmountOfDocumentsInAPartition: yacymodel.None[int](),
	}
}

func leadingWordWith(amountOfDocumentsInAPartition int) leadingword.Lead {
	return leadingword.Lead{
		Word:                          yacymodel.Some(leadingWord),
		AmountOfDocumentsInAPartition: yacymodel.Some(amountOfDocumentsInAPartition),
	}
}

func TestEachPartitionIsAskedByWhatItsDocumentsToMatchAllow(t *testing.T) {
	t.Parallel()

	documentsToMatch := documentsOf(t, 1)
	run := runWhere(
		map[uint]int{0: 1, 1: 2},
		map[uint][]yacymodel.URLHash{1: documentsToMatch},
	)

	got := asker().AskFor(roles, leadingWordWithoutAnAmount(), run)

	want := matchingwords.AsksPerPartition{
		0: matchingwords.Skipped,
		1: matchingwords.NamingTheDocumentsToMatch,
	}
	if !maps.Equal(got, want) {
		t.Fatalf("the matching words were asked %v, want %v", got, want)
	}
	if len(run.asks) != 1 || run.asks[0].partition != 1 ||
		!slices.Equal(run.asks[0].documentsToMatch, documentsToMatch) {
		t.Fatalf(
			"the asker asked %v, want one ask in partition 1 naming %v",
			run.asks, documentsToMatch,
		)
	}
}

func TestTheDocumentsToMatchAreNamedMostHeldFirst(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, 2)
	run := runWhere(map[uint]int{}, map[uint][]yacymodel.URLHash{1: documents})
	run.holdBy("first", documents...)
	run.holdBy("second", documents[1])

	asker().AskFor(roles, leadingWordWithoutAnAmount(), run)

	want := []yacymodel.URLHash{documents[1], documents[0]}
	if len(run.asks) != 1 || !slices.Equal(run.asks[0].documentsToMatch, want) {
		t.Fatalf("the asker asked %v, want one ask naming %v", run.asks, want)
	}
}

func TestAPartitionWithMoreDocumentsToMatchThanTheCeilingIsAskedWhole(t *testing.T) {
	t.Parallel()

	run := runWhere(
		map[uint]int{},
		map[uint][]yacymodel.URLHash{0: documentsOf(t, 3), 1: documentsOf(t, 1)},
	)

	got := asker().AskFor(roles, leadingWordWithoutAnAmount(), run)

	want := matchingwords.AsksPerPartition{
		0: matchingwords.OverTheCeiling,
		1: matchingwords.NamingTheDocumentsToMatch,
	}
	if !maps.Equal(got, want) || len(run.asks[0].documentsToMatch) != 0 {
		t.Fatalf("the matching words were asked %v by %v, want %v", got, run.asks, want)
	}
}

func TestAPartitionWhoseMatchingWordsWereAskedAlreadyRecordsNoKind(t *testing.T) {
	t.Parallel()

	run := runWhere(map[uint]int{}, map[uint][]yacymodel.URLHash{1: documentsOf(t, 1)})
	run.AskPartitionFor(0, roles.MatchingWords)

	got := asker().AskFor(roles, leadingWordWithoutAnAmount(), run)

	if want := (matchingwords.AsksPerPartition{1: matchingwords.NamingTheDocumentsToMatch}); !maps.Equal(
		got,
		want,
	) {
		t.Fatalf("the matching words were asked %v, want %v", got, want)
	}
}

func TestALeadingWordOverTheCeilingHasTheMatchingWordsAskedAtTheStart(
	t *testing.T,
) {
	t.Parallel()

	run := runWhere(map[uint]int{0: 1, 1: 1}, map[uint][]yacymodel.URLHash{})

	got := asker().AskFor(roles, leadingWordWith(3), run)

	want := matchingwords.AsksPerPartition{
		0: matchingwords.PredictedOverTheCeiling,
		1: matchingwords.PredictedOverTheCeiling,
	}
	if !maps.Equal(got, want) || len(run.asks) != 1 || run.asks[0].partition != everyPartition ||
		run.asks[0].waitsBefore != 0 {
		t.Fatalf(
			"the matching words were asked %v by %v, want %v in every partition before any wait",
			got, run.asks, want,
		)
	}
}

func TestALeadingWordAtTheCeilingHasTheMatchingWordsWaitForThePartition(
	t *testing.T,
) {
	t.Parallel()

	run := runWhere(map[uint]int{0: 1, 1: 1}, map[uint][]yacymodel.URLHash{1: documentsOf(t, 1)})

	asker().AskFor(roles, leadingWordWith(2), run)

	if !slices.Equal(run.waitsBeforeEachAsk(), []int{1}) {
		t.Fatalf(
			"the matching words were asked after %v waits, want one ask after its partition settled",
			run.waitsBeforeEachAsk(),
		)
	}
}
