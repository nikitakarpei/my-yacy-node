package documentmatch

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type rankedDocument struct {
	posting    yacymodel.RWIPosting
	relevance  float64
	termSpread int
}

func mergedPostingOf(postings []yacymodel.RWIPosting) yacymodel.RWIPosting {
	merged := postings[0]
	for _, posting := range postings[1:] {
		merged.TextPosition = earliestPosition(merged.TextPosition, posting.TextPosition)
		if merged.PhrasePosition > posting.PhrasePosition {
			merged.PhrasePosition = posting.PhrasePosition
			merged.PhraseRelativePosition = posting.PhraseRelativePosition
		} else if merged.PhrasePosition == posting.PhrasePosition {
			merged.PhraseRelativePosition = min(
				merged.PhraseRelativePosition,
				posting.PhraseRelativePosition,
			)
		}
		merged.TextWords = max(merged.TextWords, posting.TextWords)
		merged.TitleWords = max(merged.TitleWords, posting.TitleWords)
		merged.Phrases = max(merged.Phrases, posting.Phrases)
		merged.Hits = max(merged.Hits, posting.Hits)
	}

	return merged
}

const positionAbsent = 0

func earliestPosition(position, additional int) int {
	if position == positionAbsent || additional == positionAbsent {
		return max(position, additional)
	}

	return min(position, additional)
}

func relevanceOf(postings []yacymodel.RWIPosting, terms termsByRarity) float64 {
	relevance := 0.0
	for _, posting := range postings {
		relevance += terms.rarityOf(posting.WordHash) *
			float64(rwipostingimpactorder.ImpactOf(posting))
	}

	return relevance
}

func termSpreadOf(postings []yacymodel.RWIPosting) int {
	if len(postings) < 2 {
		return 0
	}
	firstTermPosition, lastTermPosition := postings[0].TextPosition, postings[0].TextPosition
	for _, posting := range postings[1:] {
		firstTermPosition = min(firstTermPosition, posting.TextPosition)
		lastTermPosition = max(lastTermPosition, posting.TextPosition)
	}

	return (lastTermPosition - firstTermPosition) / (len(postings) - 1)
}
