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
}

func MatchesFor(
	ctx context.Context,
	index rwipostings.PostingIndex,
	terms []yacymodel.Hash,
	filter postingfilter.Filter,
	maxPostingsPerTerm int,
) (map[yacymodel.Hash]Match, error) {
	matches := make(map[yacymodel.Hash]Match, len(terms))
	for _, term := range terms {
		postings, total, err := mostFrequentPostingsOf(
			ctx,
			index,
			term,
			filter,
			maxPostingsPerTerm,
		)
		if err != nil {
			return nil, err
		}
		matches[term] = Match{
			PostingPerDocument: postingPerDocument(postings),
			TotalMatches:       total,
		}
	}

	return matches, nil
}

func mostFrequentPostingsOf(
	ctx context.Context,
	index rwipostings.PostingIndex,
	term yacymodel.Hash,
	filter postingfilter.Filter,
	maxPostingsPerTerm int,
) ([]Posting, int, error) {
	// The per-term cap keeps the most frequent postings rather than the first
	// scanned; an exact join under a memory bound would instead pivot on the rarest term.
	frequentPostings := mostFrequentPostings{maxPostings: maxPostingsPerTerm}
	var total int
	err := index.ScanWord(ctx, term, func(posting yacymodel.RWIPosting) (bool, error) {
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
		return nil, 0, fmt.Errorf("scan term: %w", err)
	}

	return frequentPostings.postings, total, nil
}

func postingPerDocument(postings []Posting) map[yacymodel.URLHash]Posting {
	byDocument := make(map[yacymodel.URLHash]Posting, len(postings))
	for _, posting := range postings {
		byDocument[posting.DocumentHash] = posting
	}

	return byDocument
}
