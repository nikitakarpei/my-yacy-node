package searchresults_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

func TestThePageAlternatesTheSourcesByPositionTheFirstSourceFirst(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourcePage{
		resultsAt("crawled-1", "crawled-2"),
		resultsAt("network-1", "network-2"),
	})

	want := []string{"crawled-1", "network-1", "crawled-2", "network-2"}
	if got := addressesOf(page); !slices.Equal(got, want) {
		t.Errorf("addresses = %v, want %v", got, want)
	}
}

func TestAnAddressThatBothSourcesHoldShowsOnceAndRanksHigher(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourcePage{
		resultsAt("crawled-1", "shared"),
		resultsAt("shared", "network-2"),
	})

	want := []string{"shared", "crawled-1", "network-2"}
	if got := addressesOf(page); !slices.Equal(got, want) {
		t.Errorf("addresses = %v, want %v", got, want)
	}
}

func TestThePageKeepsTheResultOfTheFirstSourceThatHoldsTheAddress(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourcePage{
		{{Title: "crawled title", Address: "shared"}},
		{{Title: "network title", Address: "shared"}},
	})

	if page.Results[0].Title != "crawled title" {
		t.Errorf("title = %q, want the title of the first source", page.Results[0].Title)
	}
}

func TestThePageHasANextPageWhenASourcePageIsFull(t *testing.T) {
	fullPage := make(searchresults.SourcePage, 0, searchresults.ResultsPerSourcePage)
	for position := range searchresults.ResultsPerSourcePage {
		fullPage = append(fullPage, searchresults.Result{Address: fmt.Sprint(position)})
	}

	if !searchresults.PageFrom([]searchresults.SourcePage{resultsAt("one"), fullPage}).HasNextPage {
		t.Error("page has no next page, want one after a full source page")
	}
	if searchresults.PageFrom([]searchresults.SourcePage{resultsAt("one")}).HasNextPage {
		t.Error("page has a next page, want none after a source page that is not full")
	}
}

func TestThePageOfNoSourcePagesIsEmpty(t *testing.T) {
	page := searchresults.PageFrom(nil)

	if len(page.Results) != 0 || page.HasNextPage {
		t.Errorf("page = %+v, want an empty page", page)
	}
}

func TestTheOffsetOfAPageSkipsTheSourcePagesBeforeIt(t *testing.T) {
	if searchresults.OffsetOf(1) != 0 {
		t.Errorf("offset of page 1 = %d, want 0", searchresults.OffsetOf(1))
	}
	if searchresults.OffsetOf(3) != 2*searchresults.ResultsPerSourcePage {
		t.Errorf("offset of page 3 = %d, want two source pages", searchresults.OffsetOf(3))
	}
}

func resultsAt(addresses ...string) searchresults.SourcePage {
	page := make(searchresults.SourcePage, 0, len(addresses))
	for _, address := range addresses {
		page = append(page, searchresults.Result{Address: address})
	}
	return page
}

func addressesOf(page searchresults.Page) []string {
	addresses := make([]string, 0, len(page.Results))
	for _, result := range page.Results {
		addresses = append(addresses, result.Address)
	}
	return addresses
}
