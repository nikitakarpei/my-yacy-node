package matchingwords_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
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

type askOfTheReplicas struct {
	partition        int
	words            []yacymodel.Hash
	documentsToMatch []yacymodel.URLHash
	waitsBefore      int
}

type wordAsksOfTheReplicas struct {
	waitsUntilEachPartitionSettles map[uint]int
	documentsListedInEachPartition map[uint][]yacymodel.URLHash
	askedPartitions                map[uint]struct{}
	waits                          int
	asks                           []askOfTheReplicas
}

func wordAsksWhere(
	waitsUntilEachPartitionSettles map[uint]int,
	documentsListedInEachPartition map[uint][]yacymodel.URLHash,
) *wordAsksOfTheReplicas {
	return &wordAsksOfTheReplicas{
		waitsUntilEachPartitionSettles: waitsUntilEachPartitionSettles,
		documentsListedInEachPartition: documentsListedInEachPartition,
		askedPartitions:                map[uint]struct{}{},
	}
}

func (wordAsks *wordAsksOfTheReplicas) PartitionSettledFor(
	partition uint,
	_ []yacymodel.Hash,
) bool {
	return wordAsks.waits >= wordAsks.waitsUntilEachPartitionSettles[partition]
}

func (wordAsks *wordAsksOfTheReplicas) PartitionAskedFor(partition uint, _ []yacymodel.Hash) bool {
	_, asked := wordAsks.askedPartitions[partition]

	return asked
}

func (wordAsks *wordAsksOfTheReplicas) WaitUntilAnyPartitionSettles() {
	wordAsks.waits++
}

func (wordAsks *wordAsksOfTheReplicas) DocumentsListedIn(
	partition uint,
	_ []yacymodel.Hash,
) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	documents.AddEach(wordAsks.documentsListedInEachPartition[partition])

	return documents
}

func (wordAsks *wordAsksOfTheReplicas) AskEveryPartitionFor(words []yacymodel.Hash) {
	for partition := range uint(twoPartitionsOfTheRing) {
		wordAsks.askedPartitions[partition] = struct{}{}
	}
	wordAsks.asks = append(wordAsks.asks, askOfTheReplicas{
		partition: everyPartition, words: words, waitsBefore: wordAsks.waits,
	})
}

func (wordAsks *wordAsksOfTheReplicas) AskPartitionFor(partition uint, words []yacymodel.Hash) {
	wordAsks.AskPartitionForWordsAmong(partition, words, nil)
}

func (wordAsks *wordAsksOfTheReplicas) AskPartitionForWordsAmong(
	partition uint,
	words []yacymodel.Hash,
	documentsToMatch []yacymodel.URLHash,
) {
	wordAsks.askedPartitions[partition] = struct{}{}
	wordAsks.asks = append(wordAsks.asks, askOfTheReplicas{
		partition:        int(partition),
		words:            words,
		documentsToMatch: documentsToMatch,
		waitsBefore:      wordAsks.waits,
	})
}

func (wordAsks *wordAsksOfTheReplicas) waitsBeforeEachAsk() []int {
	waitsBeforeEachAsk := make([]int, 0, len(wordAsks.asks))
	for _, ask := range wordAsks.asks {
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

func noAmountOfDocumentsToMatch() yacymodel.Optional[int] {
	return yacymodel.None[int]()
}

func TestEachPartitionIsAskedByWhatItsDocumentsToMatchAllow(t *testing.T) {
	t.Parallel()

	documentsToMatch := documentsOf(t, 1)
	wordAsks := wordAsksWhere(
		map[uint]int{0: 1, 1: 2},
		map[uint][]yacymodel.URLHash{1: documentsToMatch},
	)

	got := asker().AskInEachPartition(roles, noAmountOfDocumentsToMatch(), wordAsks)

	want := matchingwords.KindPerPartition{
		0: matchingwords.Skipped,
		1: matchingwords.NamingTheDocumentsToMatch,
	}
	if !maps.Equal(got, want) {
		t.Fatalf("the matching words were asked %v, want %v", got, want)
	}
	if len(wordAsks.asks) != 1 || wordAsks.asks[0].partition != 1 ||
		!slices.Equal(wordAsks.asks[0].documentsToMatch, documentsToMatch) {
		t.Fatalf(
			"the asker asked %v, want one ask in partition 1 naming %v",
			wordAsks.asks, documentsToMatch,
		)
	}
}

func TestTheDocumentsToMatchAreNamedInHashOrder(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, 2)
	listed := []yacymodel.URLHash{documents[1], documents[0]}
	wordAsks := wordAsksWhere(map[uint]int{}, map[uint][]yacymodel.URLHash{1: listed})

	asker().AskInEachPartition(roles, noAmountOfDocumentsToMatch(), wordAsks)

	want := yacymodel.URLHashes{documents[0]: {}, documents[1]: {}}.InHashOrder()
	if len(wordAsks.asks) != 1 || !slices.Equal(wordAsks.asks[0].documentsToMatch, want) {
		t.Fatalf("the asker asked %v, want one ask naming %v", wordAsks.asks, want)
	}
}

func TestAPartitionWithMoreDocumentsToMatchThanTheCeilingIsAskedWhole(t *testing.T) {
	t.Parallel()

	wordAsks := wordAsksWhere(
		map[uint]int{},
		map[uint][]yacymodel.URLHash{0: documentsOf(t, 3), 1: documentsOf(t, 1)},
	)

	got := asker().AskInEachPartition(roles, noAmountOfDocumentsToMatch(), wordAsks)

	want := matchingwords.KindPerPartition{
		0: matchingwords.OverTheCeiling,
		1: matchingwords.NamingTheDocumentsToMatch,
	}
	if !maps.Equal(got, want) || len(wordAsks.asks[0].documentsToMatch) != 0 {
		t.Fatalf("the matching words were asked %v by %v, want %v", got, wordAsks.asks, want)
	}
}

func TestAPartitionWhoseMatchingWordsWereAskedAlreadyRecordsNoKind(t *testing.T) {
	t.Parallel()

	wordAsks := wordAsksWhere(map[uint]int{}, map[uint][]yacymodel.URLHash{1: documentsOf(t, 1)})
	wordAsks.AskPartitionFor(0, roles.MatchingWords)

	got := asker().AskInEachPartition(roles, noAmountOfDocumentsToMatch(), wordAsks)

	if want := (matchingwords.KindPerPartition{1: matchingwords.NamingTheDocumentsToMatch}); !maps.Equal(
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

	wordAsks := wordAsksWhere(map[uint]int{0: 1, 1: 1}, map[uint][]yacymodel.URLHash{})

	got := asker().AskInEachPartition(roles, yacymodel.Some(3), wordAsks)

	want := matchingwords.KindPerPartition{
		0: matchingwords.PredictedOverTheCeiling,
		1: matchingwords.PredictedOverTheCeiling,
	}
	if !maps.Equal(got, want) || len(wordAsks.asks) != 1 ||
		wordAsks.asks[0].partition != everyPartition ||
		wordAsks.asks[0].waitsBefore != 0 {
		t.Fatalf(
			"the matching words were asked %v by %v, want %v in every partition before any wait",
			got, wordAsks.asks, want,
		)
	}
}

func TestALeadingWordAtTheCeilingHasTheMatchingWordsWaitForThePartition(
	t *testing.T,
) {
	t.Parallel()

	wordAsks := wordAsksWhere(
		map[uint]int{0: 1, 1: 1},
		map[uint][]yacymodel.URLHash{1: documentsOf(t, 1)},
	)

	asker().AskInEachPartition(roles, yacymodel.Some(2), wordAsks)

	if !slices.Equal(wordAsks.waitsBeforeEachAsk(), []int{1}) {
		t.Fatalf(
			"the matching words were asked after %v waits, want one ask after its partition settled",
			wordAsks.waitsBeforeEachAsk(),
		)
	}
}
