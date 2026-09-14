package titletopics_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/titletopics"
)

func TestTopicsFromTitlesOrdersByFrequency(t *testing.T) {
	titles := []string{
		"alpha beta gamma",
		"alpha beta",
		"alpha",
	}

	got := titletopics.TopicsFromTitles(titles, nil)
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesExcludesQueryTerms(t *testing.T) {
	titles := []string{
		"report budget review",
		"budget review",
	}
	queryTerms := []yacymodel.Hash{yacymodel.WordHash("budget")}

	got := titletopics.TopicsFromTitles(titles, queryTerms)
	want := []string{"review", "report"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesDropsShortAndNonLetters(t *testing.T) {
	titles := []string{
		"go 2024 release notes",
		"release notes",
	}

	got := titletopics.TopicsFromTitles(titles, nil)
	want := []string{"notes", "release"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesCapsAtFiveAlphabeticallyAmongEquallyFrequentWords(t *testing.T) {
	titles := []string{
		"one two three four five six seven",
		"one two three four five six seven",
	}

	got := titletopics.TopicsFromTitles(titles, nil)
	want := []string{"five", "four", "one", "seven", "six"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesReturnsSingleWord(t *testing.T) {
	titles := []string{"alpha alpha alpha"}

	got := titletopics.TopicsFromTitles(titles, nil)
	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesDropsUnhelpfulWords(t *testing.T) {
	titles := []string{
		"the alpha and beta",
		"the alpha",
	}

	got := titletopics.TopicsFromTitles(titles, nil)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}
