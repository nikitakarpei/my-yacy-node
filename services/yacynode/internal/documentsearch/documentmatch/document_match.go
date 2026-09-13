package documentmatch

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func joinedPostingOf(postings []yacymodel.RWIPosting) yacymodel.RWIPosting {
	joined := postings[0]
	for _, posting := range postings[1:] {
		joined.TextPosition = earliestPosition(joined.TextPosition, posting.TextPosition)
		if joined.PhrasePosition > posting.PhrasePosition {
			joined.PhrasePosition = posting.PhrasePosition
			joined.PhraseRelativePosition = posting.PhraseRelativePosition
		} else if joined.PhrasePosition == posting.PhrasePosition {
			joined.PhraseRelativePosition = min(
				joined.PhraseRelativePosition,
				posting.PhraseRelativePosition,
			)
		}
		joined.TextWords = max(joined.TextWords, posting.TextWords)
		joined.TitleWords = max(joined.TitleWords, posting.TitleWords)
		joined.Phrases = max(joined.Phrases, posting.Phrases)
		joined.Hits = max(joined.Hits, posting.Hits)
	}

	return joined
}

const positionAbsent = 0

func earliestPosition(position, additional int) int {
	if position == positionAbsent || additional == positionAbsent {
		return max(position, additional)
	}

	return min(position, additional)
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
