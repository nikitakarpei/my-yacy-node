// Package documentmatch joins the per-term matches into one match per document
// and orders those documents by relevance.
package documentmatch

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type Match struct {
	documentHash      yacymodel.URLHash
	termOccurrences   int
	firstTermPosition int
	lastTermPosition  int
}

func (d Match) termSpread(termCount int) int {
	if termCount <= 1 {
		return 0
	}

	return (d.lastTermPosition - d.firstTermPosition) / (termCount - 1)
}

func MatchesAcrossEveryTerm(
	terms []yacymodel.Hash,
	matches map[yacymodel.Hash]termpostings.Match,
) map[yacymodel.URLHash]Match {
	if len(terms) == 0 {
		return nil
	}
	firstTermPostings := matches[terms[0]].PostingPerDocument
	matchingEvery := make(map[yacymodel.URLHash]Match, len(firstTermPostings))
	for documentHash, posting := range firstTermPostings {
		matchingEvery[documentHash] = Match{
			documentHash:      documentHash,
			termOccurrences:   posting.Occurrences,
			firstTermPosition: posting.TextPosition,
			lastTermPosition:  posting.TextPosition,
		}
	}

	for _, term := range terms[1:] {
		postingPerDocument := matches[term].PostingPerDocument
		for documentHash, match := range matchingEvery {
			posting, ok := postingPerDocument[documentHash]
			if !ok {
				delete(matchingEvery, documentHash)

				continue
			}
			// Deliberate divergence from YaCy, which takes the max: summing per-word
			// hit counts across the query terms ranks by total query-term frequency,
			// the relevance signal this node orders on.
			match.termOccurrences += posting.Occurrences
			match.firstTermPosition = min(match.firstTermPosition, posting.TextPosition)
			match.lastTermPosition = max(match.lastTermPosition, posting.TextPosition)
			matchingEvery[documentHash] = match
		}
	}

	return matchingEvery
}

func MatchesWithinTermSpread(
	matches map[yacymodel.URLHash]Match,
	maxTermSpread int,
	termCount int,
) map[yacymodel.URLHash]Match {
	if maxTermSpread <= 0 {
		return matches
	}
	within := make(map[yacymodel.URLHash]Match, len(matches))
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
func HashesOfMostRelevantDocuments(
	matches map[yacymodel.URLHash]Match,
	termCount int,
	maxResults int,
) []yacymodel.URLHash {
	ranked := make([]Match, 0, len(matches))
	for _, match := range matches {
		ranked = append(ranked, match)
	}
	slices.SortFunc(ranked, func(a, b Match) int {
		if a.termOccurrences != b.termOccurrences {
			return cmp.Compare(b.termOccurrences, a.termOccurrences)
		}
		if a.termSpread(termCount) != b.termSpread(termCount) {
			return cmp.Compare(a.termSpread(termCount), b.termSpread(termCount))
		}

		return cmp.Compare(a.documentHash.String(), b.documentHash.String())
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
