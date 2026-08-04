package documentsearch

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func (s searcher) scanTerm(
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

func (s searcher) excludedDocuments(
	ctx context.Context,
	terms []yacymodel.Hash,
) (map[yacymodel.URLHash]struct{}, error) {
	excludedDocuments := make(map[yacymodel.URLHash]struct{})
	for _, term := range terms {
		err := s.index.ScanWord(
			ctx,
			term,
			func(posting yacymodel.RWIPosting) (bool, error) {
				excludedDocuments[posting.URLHash] = struct{}{}

				return true, nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("scan excluded term: %w", err)
		}
	}

	return excludedDocuments, nil
}
