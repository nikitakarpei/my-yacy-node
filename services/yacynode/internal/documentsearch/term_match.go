package documentsearch

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type termMatch struct {
	postingPerDocument map[yacymodel.URLHash]termPosting
	totalPostings      int
}

func (s searcher) termMatchesFor(
	ctx context.Context,
	terms []yacymodel.Hash,
	filter postingFilter,
) (map[yacymodel.Hash]termMatch, error) {
	matches := make(map[yacymodel.Hash]termMatch, len(terms))
	for _, term := range terms {
		postings, total, err := s.mostFrequentPostingsOf(ctx, term, filter)
		if err != nil {
			return nil, err
		}
		matches[term] = termMatch{
			postingPerDocument: postingPerDocument(postings),
			totalPostings:      total,
		}
	}

	return matches, nil
}

func (s searcher) mostFrequentPostingsOf(
	ctx context.Context,
	term yacymodel.Hash,
	filter postingFilter,
) ([]termPosting, int, error) {
	// The per-term cap keeps the most frequent postings rather than the first
	// scanned; an exact join under a memory bound would instead pivot on the rarest term.
	frequentPostings := mostFrequentPostings{maxPostings: s.maxPostingsPerTerm}
	var total int
	err := s.index.ScanWord(ctx, term, func(posting yacymodel.RWIPosting) (bool, error) {
		if !filter.accepts(posting) {
			return true, nil
		}
		total++
		frequentPostings.consider(termPosting{
			documentHash: posting.URLHash,
			occurrences:  posting.Hits,
			textPosition: posting.TextPosition,
		})

		return true, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan term: %w", err)
	}

	return frequentPostings.postings, total, nil
}

func postingPerDocument(postings []termPosting) map[yacymodel.URLHash]termPosting {
	byDocument := make(map[yacymodel.URLHash]termPosting, len(postings))
	for _, posting := range postings {
		byDocument[posting.documentHash] = posting
	}

	return byDocument
}

func documentHashes(postingPerDocument map[yacymodel.URLHash]termPosting) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(postingPerDocument))
	for documentHash := range postingPerDocument {
		hashes = append(hashes, documentHash)
	}

	return hashes
}

func totalPostingsOf(matches map[yacymodel.Hash]termMatch) map[yacymodel.Hash]int {
	totals := make(map[yacymodel.Hash]int, len(matches))
	for term, match := range matches {
		totals[term] = match.totalPostings
	}

	return totals
}
