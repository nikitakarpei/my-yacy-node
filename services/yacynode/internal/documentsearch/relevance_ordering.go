package documentsearch

import (
	"maps"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type matchedDocument struct {
	identifier  yacymodel.URLHash
	occurrences int
	minPosition int
	maxPosition int
}

func (d matchedDocument) termSpread(termCount int) int {
	if termCount <= 1 {
		return 0
	}

	return (d.maxPosition - d.minPosition) / (termCount - 1)
}

func keepDocumentsMatchingEveryTerm(
	perTerm []map[yacymodel.URLHash]matchedDocument,
) map[yacymodel.URLHash]matchedDocument {
	if len(perTerm) == 0 {
		return nil
	}
	matchingEvery := make(map[yacymodel.URLHash]matchedDocument, len(perTerm[0]))
	maps.Copy(matchingEvery, perTerm[0])

	for _, documents := range perTerm[1:] {
		for identifier, document := range matchingEvery {
			alsoHere, ok := documents[identifier]
			if !ok {
				delete(matchingEvery, identifier)

				continue
			}
			// Deliberate divergence from YaCy, which takes the max: summing per-word
			// hit counts across the query terms ranks by total query-term frequency,
			// the relevance signal this node orders on.
			document.occurrences += alsoHere.occurrences
			document.minPosition = min(document.minPosition, alsoHere.minPosition)
			document.maxPosition = max(document.maxPosition, alsoHere.maxPosition)
			matchingEvery[identifier] = document
		}
	}

	return matchingEvery
}

// Deliberate divergence from YaCy: documents are ordered by occurrences and term
// spread alone, not YaCy's normalized multi-factor ranking profile. Term spread is
// the average gap between the query terms' text positions; it matches YaCy's value
// where YaCy's is deterministic, without depending on YaCy's join-order-sensitive
// position queue.
func documentsOrderedByRelevance(
	documents map[yacymodel.URLHash]matchedDocument,
	termCount int,
) []yacymodel.URLHash {
	ranked := make([]matchedDocument, 0, len(documents))
	for _, document := range documents {
		ranked = append(ranked, document)
	}
	slices.SortFunc(ranked, func(a, b matchedDocument) int {
		if a.occurrences != b.occurrences {
			return compareDescending(a.occurrences, b.occurrences)
		}
		if a.termSpread(termCount) != b.termSpread(termCount) {
			return compareAscending(a.termSpread(termCount), b.termSpread(termCount))
		}

		return compareAscending(a.identifier.String(), b.identifier.String())
	})

	identifiers := make([]yacymodel.URLHash, 0, len(ranked))
	for _, document := range ranked {
		identifiers = append(identifiers, document.identifier)
	}

	return identifiers
}

func documentsWithinTermSpread(
	documents map[yacymodel.URLHash]matchedDocument,
	maxTermSpread int,
	termCount int,
) map[yacymodel.URLHash]matchedDocument {
	if maxTermSpread <= 0 {
		return documents
	}
	for identifier, document := range documents {
		if document.termSpread(termCount) > maxTermSpread {
			delete(documents, identifier)
		}
	}

	return documents
}

func takeMostRelevant(identifiers []yacymodel.URLHash, limit int) []yacymodel.URLHash {
	if limit > 0 && len(identifiers) > limit {
		return identifiers[:limit]
	}

	return identifiers
}

func documentIdentifiers(documents map[yacymodel.URLHash]matchedDocument) []yacymodel.URLHash {
	identifiers := make([]yacymodel.URLHash, 0, len(documents))
	for identifier := range documents {
		identifiers = append(identifiers, identifier)
	}

	return identifiers
}

func compareDescending[T ~int](a, b T) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}

func compareAscending[T ~int | ~string](a, b T) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
