// Package searchresults merges the result pages that the sources of one search give into the
// page that a person sees. A result scores one over its position in each source page that holds
// its address. The page shows each address once, the highest score first.
package searchresults

import (
	"cmp"
	"slices"
)

const ResultsPerSourcePage = 10

type Result struct {
	Title   string
	Address string
	Snippet string
}

type SourcePage []Result

type Page struct {
	Results     []Result
	HasNextPage bool
}

func OffsetOf(pageNumber int) int {
	return (pageNumber - 1) * ResultsPerSourcePage
}

func PageFrom(sourcePages []SourcePage) Page {
	scores := map[string]float64{}
	resultsByAddress := map[string]Result{}
	var addresses []string
	hasNextPage := false
	for _, sourcePage := range sourcePages {
		hasNextPage = hasNextPage || len(sourcePage) >= ResultsPerSourcePage
		for index, result := range sourcePage {
			if _, seen := resultsByAddress[result.Address]; !seen {
				resultsByAddress[result.Address] = result
				addresses = append(addresses, result.Address)
			}
			scores[result.Address] += 1 / float64(index+1)
		}
	}
	slices.SortStableFunc(addresses, func(first, second string) int {
		return cmp.Compare(scores[second], scores[first])
	})
	results := make([]Result, 0, len(addresses))
	for _, address := range addresses {
		results = append(results, resultsByAddress[address])
	}
	return Page{Results: results, HasNextPage: hasNextPage}
}
