package documentsearch

import (
	"context"

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
		postings, total, err := s.scanTerm(ctx, term, filter)
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
