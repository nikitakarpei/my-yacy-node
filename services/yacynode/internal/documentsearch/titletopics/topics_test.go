package titletopics_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/titletopics"
)

func titled(title string) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{Title: title}
}

func TestTopicsFromTitlesOrdersByFrequency(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("alpha beta gamma"),
		titled("alpha beta"),
		titled("alpha"),
	}

	got := titletopics.TopicsFromTitles(resources, nil)
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesExcludesQueryTerms(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("report budget review"),
		titled("budget review"),
	}
	queryTerms := []yacymodel.Hash{yacymodel.WordHash("budget")}

	got := titletopics.TopicsFromTitles(resources, queryTerms)
	want := []string{"review", "report"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesDropsShortAndNonLetters(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("go 2024 release notes"),
		titled("release notes"),
	}

	got := titletopics.TopicsFromTitles(resources, nil)
	want := []string{"notes", "release"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesCapsAtFiveAlphabeticallyAmongEquallyFrequentWords(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("one two three four five six seven"),
		titled("one two three four five six seven"),
	}

	got := titletopics.TopicsFromTitles(resources, nil)
	want := []string{"five", "four", "one", "seven", "six"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesReturnsSingleWord(t *testing.T) {
	resources := []yacymodel.URLMetadata{titled("alpha alpha alpha")}

	got := titletopics.TopicsFromTitles(resources, nil)
	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestTopicsFromTitlesDropsUnhelpfulWords(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("the alpha and beta"),
		titled("the alpha"),
	}

	got := titletopics.TopicsFromTitles(resources, nil)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}
