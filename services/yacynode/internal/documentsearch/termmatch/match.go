// Package termmatch collects, for each query term, the documents that hold the
// term and how often the term appears in them.
package termmatch

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Match struct {
	PostingPerDocument map[yacymodel.URLHash]Posting
	TotalMatches       int
	PostingsHeld       int
}

func MatchesFor(
	ctx context.Context,
	terms []yacymodel.Hash,
	index rwipostings.PostingIndex,
	filter postingfilter.Filter,
	maxPostingsPerTerm int,
) (map[yacymodel.Hash]Match, error) {
	matches := make(map[yacymodel.Hash]Match, len(terms))
	for _, term := range terms {
		match, err := matchOf(ctx, term, index, filter, maxPostingsPerTerm)
		if err != nil {
			return nil, err
		}
		matches[term] = match
	}

	return matches, nil
}

func matchOf(
	ctx context.Context,
	term yacymodel.Hash,
	index rwipostings.PostingIndex,
	filter postingfilter.Filter,
	maxPostingsPerTerm int,
) (Match, error) {
	// The per-term cap keeps the most frequent postings rather than the first
	// scanned; an exact join under a memory bound would instead pivot on the rarest term.
	frequentPostings := mostFrequentPostings{maxPostings: maxPostingsPerTerm}
	var held, total int
	err := index.ScanWord(ctx, term, func(posting yacymodel.RWIPosting) (bool, error) {
		held++
		if !filter.Accepts(posting) {
			return true, nil
		}
		total++
		frequentPostings.consider(Posting{
			DocumentHash: posting.URLHash,
			Occurrences:  posting.Hits,
			TextPosition: posting.TextPosition,
		})

		return true, nil
	})
	if err != nil {
		return Match{}, fmt.Errorf("scan term: %w", err)
	}

	return Match{
		PostingPerDocument: postingPerDocument(frequentPostings.postings),
		TotalMatches:       total,
		PostingsHeld:       held,
	}, nil
}

func postingPerDocument(postings []Posting) map[yacymodel.URLHash]Posting {
	byDocument := make(map[yacymodel.URLHash]Posting, len(postings))
	for _, posting := range postings {
		byDocument[posting.DocumentHash] = posting
	}

	return byDocument
}
