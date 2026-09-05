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
	posting           yacymodel.RWIPosting
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
			posting:           posting,
			termOccurrences:   posting.Hits,
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
			match.posting = postingAcrossTerms(match.posting, posting)
			match.termOccurrences += posting.Hits
			match.firstTermPosition = min(match.firstTermPosition, posting.TextPosition)
			match.lastTermPosition = max(match.lastTermPosition, posting.TextPosition)
			matchingEvery[documentHash] = match
		}
	}

	return matchingEvery
}

// postingAcrossTerms merges the postings of one document the way YaCy's own join
// does, so the posting this node hands a peer is the posting that peer would
// have built for itself: the earliest positions, and the largest counts.
func postingAcrossTerms(posting, additional yacymodel.RWIPosting) yacymodel.RWIPosting {
	posting.TextPosition = earliestPosition(posting.TextPosition, additional.TextPosition)
	if posting.PhrasePosition > additional.PhrasePosition {
		posting.PhrasePosition = additional.PhrasePosition
		posting.PhraseRelativePosition = additional.PhraseRelativePosition
	} else if posting.PhrasePosition == additional.PhrasePosition {
		posting.PhraseRelativePosition = min(
			posting.PhraseRelativePosition,
			additional.PhraseRelativePosition,
		)
	}
	posting.TextWords = max(posting.TextWords, additional.TextWords)
	posting.TitleWords = max(posting.TitleWords, additional.TitleWords)
	posting.Phrases = max(posting.Phrases, additional.Phrases)
	posting.Hits = max(posting.Hits, additional.Hits)

	return posting
}

func earliestPosition(position, additional int) int {
	if position == 0 || additional == 0 {
		return max(position, additional)
	}

	return min(position, additional)
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
func PostingsOfMostRelevantDocuments(
	matches map[yacymodel.URLHash]Match,
	termCount int,
	maxResults int,
) []yacymodel.RWIPosting {
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

		return cmp.Compare(a.posting.URLHash.String(), b.posting.URLHash.String())
	})

	postings := make([]yacymodel.RWIPosting, 0, len(ranked))
	for _, match := range ranked {
		postings = append(postings, match.posting)
	}
	if maxResults > 0 && len(postings) > maxResults {
		return postings[:maxResults]
	}

	return postings
}
