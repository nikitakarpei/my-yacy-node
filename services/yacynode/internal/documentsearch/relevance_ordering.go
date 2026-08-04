package documentsearch

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentMatch struct {
	documentHash      yacymodel.URLHash
	termOccurrences   int
	firstTermPosition int
	lastTermPosition  int
}

func (d documentMatch) termSpread(termCount int) int {
	if termCount <= 1 {
		return 0
	}

	return (d.lastTermPosition - d.firstTermPosition) / (termCount - 1)
}

func documentMatchesAcrossEveryTerm(
	terms []yacymodel.Hash,
	matches map[yacymodel.Hash]termMatch,
) map[yacymodel.URLHash]documentMatch {
	if len(terms) == 0 {
		return nil
	}
	firstTermPostings := matches[terms[0]].postingPerDocument
	matchingEvery := make(map[yacymodel.URLHash]documentMatch, len(firstTermPostings))
	for documentHash, posting := range firstTermPostings {
		matchingEvery[documentHash] = documentMatch{
			documentHash:      documentHash,
			termOccurrences:   posting.occurrences,
			firstTermPosition: posting.textPosition,
			lastTermPosition:  posting.textPosition,
		}
	}

	for _, term := range terms[1:] {
		postingPerDocument := matches[term].postingPerDocument
		for documentHash, match := range matchingEvery {
			posting, ok := postingPerDocument[documentHash]
			if !ok {
				delete(matchingEvery, documentHash)

				continue
			}
			// Deliberate divergence from YaCy, which takes the max: summing per-word
			// hit counts across the query terms ranks by total query-term frequency,
			// the relevance signal this node orders on.
			match.termOccurrences += posting.occurrences
			match.firstTermPosition = min(match.firstTermPosition, posting.textPosition)
			match.lastTermPosition = max(match.lastTermPosition, posting.textPosition)
			matchingEvery[documentHash] = match
		}
	}

	return matchingEvery
}

func documentMatchesWithinTermSpread(
	matches map[yacymodel.URLHash]documentMatch,
	maxTermSpread int,
	termCount int,
) map[yacymodel.URLHash]documentMatch {
	if maxTermSpread <= 0 {
		return matches
	}
	within := make(map[yacymodel.URLHash]documentMatch, len(matches))
	for documentHash, match := range matches {
		if match.termSpread(termCount) <= maxTermSpread {
			within[documentHash] = match
		}
	}

	return within
}

// Deliberate divergence from YaCy: documents are ordered by occurrences and term
// spread alone, not YaCy's normalized multi-factor ranking profile. Term spread is
// the average gap between the query terms' text positions; it matches YaCy's value
// where YaCy's is deterministic, without depending on YaCy's join-order-sensitive
// position queue.
func hashesOfMostRelevantDocuments(
	matches map[yacymodel.URLHash]documentMatch,
	termCount int,
	maxResults int,
) []yacymodel.URLHash {
	ranked := make([]documentMatch, 0, len(matches))
	for _, match := range matches {
		ranked = append(ranked, match)
	}
	slices.SortFunc(ranked, func(a, b documentMatch) int {
		if a.termOccurrences != b.termOccurrences {
			return compareDescending(a.termOccurrences, b.termOccurrences)
		}
		if a.termSpread(termCount) != b.termSpread(termCount) {
			return compareAscending(a.termSpread(termCount), b.termSpread(termCount))
		}

		return compareAscending(a.documentHash.String(), b.documentHash.String())
	})

	documentHashes := make([]yacymodel.URLHash, 0, len(ranked))
	for _, match := range ranked {
		documentHashes = append(documentHashes, match.documentHash)
	}
	if maxResults > 0 && len(documentHashes) > maxResults {
		return documentHashes[:maxResults]
	}

	return documentHashes
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
