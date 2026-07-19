package documentsearch

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func titled(title string) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{Title: title}
}

func TestResultTopicsOrdersByFrequency(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("alpha beta gamma"),
		titled("alpha beta"),
		titled("alpha"),
	}

	got := resultTopics(resources, nil)
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestResultTopicsExcludesQueryTerms(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("report budget review"),
		titled("budget review"),
	}
	queryTerms := []yacymodel.Hash{yacymodel.WordHash("budget")}

	got := resultTopics(resources, queryTerms)
	want := []string{"review", "report"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestResultTopicsDropsShortAndNonLetters(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("go 2024 release notes"),
		titled("release notes"),
	}

	got := resultTopics(resources, nil)
	want := []string{"notes", "release"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestResultTopicsCapsAtFive(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("one two three four five six seven"),
		titled("one two three four five six seven"),
	}

	got := resultTopics(resources, nil)
	if len(got) != maxTopics {
		t.Fatalf("topic count = %d, want %d", len(got), maxTopics)
	}
}

func TestResultTopicsReturnsSingleWord(t *testing.T) {
	resources := []yacymodel.URLMetadata{titled("alpha alpha alpha")}

	got := resultTopics(resources, nil)
	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}

func TestResultTopicsDropsUnhelpfulWords(t *testing.T) {
	resources := []yacymodel.URLMetadata{
		titled("the alpha and beta"),
		titled("the alpha"),
	}

	got := resultTopics(resources, nil)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("topics = %v, want %v", got, want)
	}
}
