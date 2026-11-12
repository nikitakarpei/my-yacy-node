package searchresults_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

func TestThePageAlternatesTheSourcesByPositionTheFirstSourceFirst(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourceRanking{
		resultsAt("crawled-1", "crawled-2"),
		resultsAt("network-1", "network-2"),
	}, 1)

	want := []string{"crawled-1", "network-1", "crawled-2", "network-2"}
	if got := addressesOf(page); !slices.Equal(got, want) {
		t.Errorf("addresses = %v, want %v", got, want)
	}
}

func TestAnAddressThatBothSourcesHoldShowsOnceAndRanksHigher(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourceRanking{
		resultsAt("crawled-1", "shared"),
		resultsAt("shared", "network-2"),
	}, 1)

	want := []string{"shared", "crawled-1", "network-2"}
	if got := addressesOf(page); !slices.Equal(got, want) {
		t.Errorf("addresses = %v, want %v", got, want)
	}
}

func TestAnAddressThatBothSourcesHoldLowerRanksAboveTheFirstOfOneSource(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourceRanking{
		resultsAt("crawled-1", "shared"),
		resultsAt("network-1", "shared"),
	}, 1)

	want := []string{"shared", "crawled-1", "network-1"}
	if got := addressesOf(page); !slices.Equal(got, want) {
		t.Errorf("addresses = %v, want %v", got, want)
	}
}

func TestThePageKeepsTheResultOfTheFirstSourceThatHoldsTheAddress(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourceRanking{
		{{Title: "crawled title", Address: "shared"}},
		{{Title: "network title", Address: "shared"}},
	}, 1)

	if page.Results[0].Title != "crawled title" {
		t.Errorf("title = %q, want the title of the first source", page.Results[0].Title)
	}
}

func TestALaterPageContinuesTheMergedListOfTheEarlierPages(t *testing.T) {
	sourceRankings := []searchresults.SourceRanking{
		rankingOf("crawled", searchresults.DepthOf(2)),
		rankingOf("network", searchresults.ResultsPerPage/2),
	}

	firstPage := addressesOf(searchresults.PageFrom(sourceRankings, 1))
	secondPage := addressesOf(searchresults.PageFrom(sourceRankings, 2))

	if len(firstPage) != searchresults.ResultsPerPage ||
		len(secondPage) != searchresults.ResultsPerPage {
		t.Fatalf("pages = %v and %v, want two full pages", firstPage, secondPage)
	}
	for _, address := range secondPage {
		if slices.Contains(firstPage, address) {
			t.Errorf("address %s shows on both pages", address)
		}
	}
	if !slices.Contains(secondPage, "crawled-11") {
		t.Errorf("second page = %v, want the crawled results that the first page left", secondPage)
	}
}

func TestThePageHasANextPageWhenASourceReachesTheDepthOfThePage(t *testing.T) {
	fullRanking := rankingOf("network", searchresults.DepthOf(1))

	if !searchresults.PageFrom([]searchresults.SourceRanking{resultsAt("one"), fullRanking}, 1).
		HasNextPage {
		t.Error("page has no next page, want one after a source that reaches the depth")
	}
	if searchresults.PageFrom([]searchresults.SourceRanking{resultsAt("one")}, 1).HasNextPage {
		t.Error("page has a next page, want none after a source that ends before the depth")
	}
}

func TestThePageHasANextPageWhenTheMergedListGoesOnPastIt(t *testing.T) {
	page := searchresults.PageFrom([]searchresults.SourceRanking{
		rankingOf("crawled", searchresults.ResultsPerPage-1),
		rankingOf("network", searchresults.ResultsPerPage-1),
	}, 1)

	if !page.HasNextPage {
		t.Error("page has no next page, want one for the merged results past the page")
	}
}

func TestAPagePastTheLastPageIsEmpty(t *testing.T) {
	const farPageNumber = 1000
	page := searchresults.PageFrom([]searchresults.SourceRanking{
		rankingOf("crawled", searchresults.DepthOf(farPageNumber)),
	}, farPageNumber)

	if len(page.Results) != 0 || page.HasNextPage {
		t.Errorf("page = %+v, want an empty page", page)
	}
}

func TestTheLastPageHasNoNextPage(t *testing.T) {
	sourceRankings := []searchresults.SourceRanking{rankingOf("crawled", searchresults.DepthOf(1))}
	pageNumber := 1
	for searchresults.PageFrom(sourceRankings, pageNumber).HasNextPage {
		pageNumber++
		sourceRankings = []searchresults.SourceRanking{
			rankingOf("crawled", searchresults.DepthOf(pageNumber)),
		}
	}

	if searchresults.DepthOf(pageNumber+1) != searchresults.DepthOf(pageNumber) {
		t.Errorf("page %d has no next page, want the depth to stop growing there", pageNumber)
	}
}

func TestThePageOfNoSourceRankingsIsEmpty(t *testing.T) {
	page := searchresults.PageFrom(nil, 1)

	if len(page.Results) != 0 || page.HasNextPage {
		t.Errorf("page = %+v, want an empty page", page)
	}
}

func TestTheDepthOfAPageReachesTheEndOfThePage(t *testing.T) {
	if searchresults.DepthOf(1) != searchresults.ResultsPerPage {
		t.Errorf("depth of page 1 = %d, want one page", searchresults.DepthOf(1))
	}
	if searchresults.DepthOf(3) != 3*searchresults.ResultsPerPage {
		t.Errorf("depth of page 3 = %d, want three pages", searchresults.DepthOf(3))
	}
}

func resultsAt(addresses ...string) searchresults.SourceRanking {
	ranking := make(searchresults.SourceRanking, 0, len(addresses))
	for _, address := range addresses {
		ranking = append(ranking, searchresults.Result{Address: address})
	}
	return ranking
}

func rankingOf(source string, amountOfResults int) searchresults.SourceRanking {
	ranking := make(searchresults.SourceRanking, 0, amountOfResults)
	for position := 1; position <= amountOfResults; position++ {
		ranking = append(
			ranking,
			searchresults.Result{Address: fmt.Sprintf("%s-%d", source, position)},
		)
	}
	return ranking
}

func addressesOf(page searchresults.Page) []string {
	addresses := make([]string, 0, len(page.Results))
	for _, result := range page.Results {
		addresses = append(addresses, result.Address)
	}
	return addresses
}
