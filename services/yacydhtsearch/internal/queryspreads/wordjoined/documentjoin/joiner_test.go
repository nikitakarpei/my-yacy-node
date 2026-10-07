package documentjoin_test

import (
	"maps"
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentjoin"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord  = "berlin"
	secondWord = "weather"
)

var query = queryreading.QueryFrom(firstWord+" "+secondWord, yacymodel.Language{})

func documentHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("%q has no document hash: %v", address, err)
	}

	return hash
}

type joinsInTurn struct {
	joined []yacymodel.URLHashes
}

func (joins *joinsInTurn) DocumentsJoined(documents yacymodel.URLHashes) {
	joins.joined = append(joins.joined, documents)
}

func joinerOver(joins *joinsInTurn) *documentjoin.Joiner {
	return documentjoin.JoinerOf(query, joins)
}

func documentsOf(documents ...yacymodel.URLHash) yacymodel.URLHashes {
	listedDocuments := yacymodel.URLHashes{}
	listedDocuments.AddEach(documents)

	return listedDocuments
}

func TestADocumentJoinsOnTheListingOfTheLastQueryWordThatListsIt(t *testing.T) {
	t.Parallel()

	listedByBoth := documentHashOf(t, "https://listed-by-both.example/")
	listedByOne := documentHashOf(t, "https://listed-by-one.example/")
	joins := &joinsInTurn{}
	joiner := joinerOver(joins)

	joiner.ListUnder(yacymodel.WordHash(firstWord), documentsOf(listedByOne, listedByBoth))
	joinedByTheFirstWord := len(joins.joined)
	joiner.ListUnder(yacymodel.WordHash(secondWord), documentsOf(listedByBoth))

	if want := []yacymodel.URLHashes{documentsOf(listedByBoth)}; joinedByTheFirstWord != 0 ||
		!reflect.DeepEqual(joins.joined, want) {
		t.Fatalf("the listings joined %v, want nothing then %v", joins.joined, want)
	}
}

func TestTheListingOfACompoundWordJoinsForEachOfItsPartWords(t *testing.T) {
	t.Parallel()

	document := documentHashOf(t, "https://document.example/")
	joins := &joinsInTurn{}
	joiner := joinerOver(joins)

	joiner.ListUnder(yacymodel.WordHash(firstWord+secondWord), documentsOf(document))

	if want := []yacymodel.URLHashes{
		documentsOf(document),
	}; !reflect.DeepEqual(
		joins.joined,
		want,
	) {
		t.Fatalf("the listing of the compound word joined %v, want %v", joins.joined, want)
	}
}

func TestADocumentJoinsOnce(t *testing.T) {
	t.Parallel()

	document := documentHashOf(t, "https://document.example/")
	joins := &joinsInTurn{}
	joiner := joinerOver(joins)
	joiner.ListUnder(yacymodel.WordHash(firstWord), documentsOf(document))
	joiner.ListUnder(yacymodel.WordHash(secondWord), documentsOf(document))

	joiner.ListUnder(yacymodel.WordHash(secondWord), documentsOf(document))

	if len(joins.joined) != 1 {
		t.Fatalf("the listings joined %v, want the document once", joins.joined)
	}
}

func TestTheListingOfAWordOutsideTheQueryJoinsNothing(t *testing.T) {
	t.Parallel()

	document := documentHashOf(t, "https://document.example/")
	joins := &joinsInTurn{}
	joiner := joinerOver(joins)
	joiner.ListUnder(yacymodel.WordHash(firstWord), documentsOf(document))

	joiner.ListUnder(yacymodel.WordHash("rain"), documentsOf(document))

	if len(joins.joined) != 0 || len(joiner.JoinedDocuments()) != 0 {
		t.Fatalf(
			"the listing of a word outside the query joined %v, and the join holds %v, "+
				"want nothing",
			joins.joined, joiner.JoinedDocuments(),
		)
	}
}

func TestTheJoinedDocumentsAreEveryDocumentJoinedSoFar(t *testing.T) {
	t.Parallel()

	joinedFirst := documentHashOf(t, "https://joined-first.example/")
	joinedLater := documentHashOf(t, "https://joined-later.example/")
	joiner := joinerOver(&joinsInTurn{})
	joiner.ListUnder(yacymodel.WordHash(firstWord), documentsOf(joinedFirst, joinedLater))
	joiner.ListUnder(yacymodel.WordHash(secondWord), documentsOf(joinedFirst))
	joiner.ListUnder(yacymodel.WordHash(firstWord+secondWord), documentsOf(joinedLater))

	if got, want := joiner.JoinedDocuments(), documentsOf(joinedFirst, joinedLater); !maps.Equal(
		got, want,
	) {
		t.Fatalf("the join holds %v, want %v", got, want)
	}
}

func TestEveryJoinObserverIsToldTheJoinedDocuments(t *testing.T) {
	t.Parallel()

	document := documentHashOf(t, "https://document.example/")
	first, second := &joinsInTurn{}, &joinsInTurn{}
	joiner := documentjoin.JoinerOf(query, documentjoin.JoinObservers{first, second})

	joiner.ListUnder(yacymodel.WordHash(firstWord+secondWord), documentsOf(document))

	if want := []yacymodel.URLHashes{
		documentsOf(document),
	}; !reflect.DeepEqual(
		first.joined,
		want,
	) ||
		!reflect.DeepEqual(second.joined, want) {
		t.Fatalf(
			"the observers were told %v and %v, want both told %v",
			first.joined,
			second.joined,
			want,
		)
	}
}
