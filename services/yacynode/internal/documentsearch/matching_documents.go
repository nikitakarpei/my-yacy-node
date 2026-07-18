package documentsearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type termMatches struct {
	documentsPerTerm    map[yacymodel.Hash]map[yacymodel.Hash]matchedDocument
	totalMatchesPerTerm map[yacymodel.Hash]int
}

func (s searcher) documentsMatchingTerms(
	ctx context.Context,
	terms []yacymodel.Hash,
	filter postingFilter,
) (termMatches, error) {
	matches := termMatches{
		documentsPerTerm: make(
			map[yacymodel.Hash]map[yacymodel.Hash]matchedDocument,
			len(terms),
		),
		totalMatchesPerTerm: make(map[yacymodel.Hash]int, len(terms)),
	}
	for _, term := range terms {
		postings, total, err := s.scanTerm(ctx, term, filter)
		if err != nil {
			return termMatches{}, err
		}
		matches.documentsPerTerm[term] = dedupeDocuments(postings)
		matches.totalMatchesPerTerm[term] = total
	}

	return matches, nil
}

func dedupeDocuments(postings []termPosting) map[yacymodel.Hash]matchedDocument {
	documents := make(map[yacymodel.Hash]matchedDocument, len(postings))
	for _, posting := range postings {
		if _, seen := documents[posting.documentIdentifier]; seen {
			continue
		}
		documents[posting.documentIdentifier] = matchedDocument{
			identifier:  posting.documentIdentifier,
			occurrences: posting.occurrences,
			minPosition: posting.textPosition,
			maxPosition: posting.textPosition,
		}
	}

	return documents
}

func documentsInTermOrder(
	terms []yacymodel.Hash,
	documentsPerTerm map[yacymodel.Hash]map[yacymodel.Hash]matchedDocument,
) []map[yacymodel.Hash]matchedDocument {
	ordered := make([]map[yacymodel.Hash]matchedDocument, 0, len(terms))
	for _, term := range terms {
		ordered = append(ordered, documentsPerTerm[term])
	}

	return ordered
}
