package documentmatch

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type documentMatch struct {
	posting           yacymodel.RWIPosting
	relevance         float64
	firstTermPosition int
	lastTermPosition  int
}

func matchAcrossTerms(postings []yacymodel.RWIPosting, rarity wordRarity) documentMatch {
	match := documentMatch{
		posting:           postings[0],
		relevance:         relevanceOf(postings[0], rarity),
		firstTermPosition: postings[0].TextPosition,
		lastTermPosition:  postings[0].TextPosition,
	}
	for _, posting := range postings[1:] {
		match.posting = postingAcrossTerms(match.posting, posting)
		match.relevance += relevanceOf(posting, rarity)
		match.firstTermPosition = min(match.firstTermPosition, posting.TextPosition)
		match.lastTermPosition = max(match.lastTermPosition, posting.TextPosition)
	}

	return match
}

func relevanceOf(posting yacymodel.RWIPosting, rarity wordRarity) float64 {
	return rarity.rarityOf(posting.WordHash) * float64(rwipostingimpactorder.ImpactOf(posting))
}

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

const positionAbsent = 0

func earliestPosition(position, additional int) int {
	if position == positionAbsent || additional == positionAbsent {
		return max(position, additional)
	}

	return min(position, additional)
}

func (d documentMatch) termSpread(amountOfTerms int) int {
	if amountOfTerms <= 1 {
		return 0
	}

	return (d.lastTermPosition - d.firstTermPosition) / (amountOfTerms - 1)
}
