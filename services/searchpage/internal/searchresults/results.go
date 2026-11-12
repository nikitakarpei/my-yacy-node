// Package searchresults merges the rankings that the sources of one search give into the page
// that a person sees. Each source ranks from its first result down to the depth of the page. A
// result scores by reciprocal rank fusion over the sources that hold its address, and the merged
// list shows each address once, the highest score first. A page is one cut of that list.
package searchresults

import (
	"cmp"
	"slices"
)

const (
	ResultsPerPage = 10
	lastPageNumber = 5
	rankConstant   = 60
)

type Result struct {
	Title   string
	Address string
	Snippet string
}

type SourceRanking []Result

type Page struct {
	Results     []Result
	HasNextPage bool
}

func DepthOf(pageNumber int) int {
	return min(pageNumber, lastPageNumber) * ResultsPerPage
}

func PageFrom(sourceRankings []SourceRanking, pageNumber int) Page {
	if pageNumber > lastPageNumber {
		return Page{}
	}
	mergedResults := mergedFrom(sourceRankings)
	start := min((pageNumber-1)*ResultsPerPage, len(mergedResults))
	end := min(start+ResultsPerPage, len(mergedResults))
	return Page{
		Results: mergedResults[start:end],
		HasNextPage: pageNumber < lastPageNumber &&
			(len(mergedResults) > end || anyReaches(sourceRankings, DepthOf(pageNumber))),
	}
}

func mergedFrom(sourceRankings []SourceRanking) []Result {
	scores := map[string]float64{}
	resultsByAddress := map[string]Result{}
	var addresses []string
	for _, sourceRanking := range sourceRankings {
		for index, result := range sourceRanking {
			if _, seen := resultsByAddress[result.Address]; !seen {
				resultsByAddress[result.Address] = result
				addresses = append(addresses, result.Address)
			}
			scores[result.Address] += 1 / float64(rankConstant+index+1)
		}
	}
	slices.SortStableFunc(addresses, func(first, second string) int {
		return cmp.Compare(scores[second], scores[first])
	})
	mergedResults := make([]Result, 0, len(addresses))
	for _, address := range addresses {
		mergedResults = append(mergedResults, resultsByAddress[address])
	}
	return mergedResults
}

func anyReaches(sourceRankings []SourceRanking, depth int) bool {
	return slices.ContainsFunc(sourceRankings, func(sourceRanking SourceRanking) bool {
		return len(sourceRanking) >= depth
	})
}
