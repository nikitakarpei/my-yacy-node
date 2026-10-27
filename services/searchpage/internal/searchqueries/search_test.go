package searchqueries_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
)

func TestASearchWithoutOperatorsAsksEverySourceInAllLanguages(t *testing.T) {
	search := searchqueries.SearchFrom("  peer to   peer ", "", searchqueries.Sources())

	if search.Text() != "peer to peer" {
		t.Errorf("text = %q, want the query words", search.Text())
	}
	if !slices.Equal(search.Sources, searchqueries.Sources()) {
		t.Errorf("sources = %v, want every source", search.Sources)
	}
	if search.Language != "" {
		t.Errorf("language = %q, want all languages", search.Language)
	}
	if !search.IsFirstPage() || search.PageNumber != 1 {
		t.Errorf("page number = %d, want the first page", search.PageNumber)
	}
}

func TestASourceOperatorChoosesItsSourceAndLeavesTheQueryWords(t *testing.T) {
	search := searchqueries.SearchFrom("!ya yacy", "", searchqueries.Sources())

	if !slices.Equal(search.Sources, []searchqueries.Source{searchqueries.YaCyNetwork}) {
		t.Errorf("sources = %v, want only the YaCy network", search.Sources)
	}
	if search.Text() != "yacy" {
		t.Errorf("text = %q, want the word without the operator", search.Text())
	}
}

func TestTwoSourceOperatorsChooseBothSourcesInTheirOwnOrder(t *testing.T) {
	search := searchqueries.SearchFrom("!ya !ct yacy", "", searchqueries.Sources())

	if !slices.Equal(search.Sources, searchqueries.Sources()) {
		t.Errorf("sources = %v, want every source in order", search.Sources)
	}
}

func TestAnUnknownSourceOperatorStaysAQueryWord(t *testing.T) {
	search := searchqueries.SearchFrom("!xyz yacy", "", searchqueries.Sources())

	if search.Text() != "!xyz yacy" {
		t.Errorf("text = %q, want the unknown operator kept", search.Text())
	}
	if !slices.Equal(search.Sources, searchqueries.Sources()) {
		t.Errorf("sources = %v, want every source", search.Sources)
	}
}

func TestTheShortcutOfASourceThatIsNotOfferedStaysAQueryWord(t *testing.T) {
	offeredSources := []searchqueries.Source{searchqueries.YaCyNetwork}

	search := searchqueries.SearchFrom("!ct yacy", "", offeredSources)

	if search.Text() != "!ct yacy" {
		t.Errorf("text = %q, want the shortcut kept", search.Text())
	}
	if !slices.Equal(search.Sources, offeredSources) {
		t.Errorf("sources = %v, want only the offered source", search.Sources)
	}
}

func TestALanguageOperatorChoosesTheLanguageInLowerCase(t *testing.T) {
	search := searchqueries.SearchFrom("yacy :DE", "", searchqueries.Sources())

	if search.Language != "de" {
		t.Errorf("language = %q, want de", search.Language)
	}
	if search.Text() != "yacy" {
		t.Errorf("text = %q, want the word without the operator", search.Text())
	}
}

func TestAWordThatOnlyLooksLikeALanguageOperatorStaysAQueryWord(t *testing.T) {
	for _, word := range []string{":deu", ":d1", ":"} {
		search := searchqueries.SearchFrom("yacy "+word, "", searchqueries.Sources())

		if search.Language != "" || search.Text() != "yacy "+word {
			t.Errorf("%q gave language %q and text %q, want a query word",
				word, search.Language, search.Text())
		}
	}
}

func TestAQueryOfOnlyOperatorsHasNoWords(t *testing.T) {
	if searchqueries.SearchFrom("!ct :de", "", searchqueries.Sources()).HasWords() {
		t.Error("search has words, want none")
	}
	if !searchqueries.SearchFrom("yacy", "", searchqueries.Sources()).HasWords() {
		t.Error("search has no words, want one")
	}
}

func TestThePageNumberIsTheAskedPageOrElseTheFirst(t *testing.T) {
	for pageNumber, want := range map[string]int{"3": 3, "0": 1, "-2": 1, "two": 1, "": 1} {
		search := searchqueries.SearchFrom("yacy", pageNumber, searchqueries.Sources())

		if search.PageNumber != want {
			t.Errorf("page %q gave page number %d, want %d", pageNumber, search.PageNumber, want)
		}
	}
	if searchqueries.SearchFrom("yacy", "3", searchqueries.Sources()).IsFirstPage() {
		t.Error("page 3 is the first page, want a later page")
	}
}
