package wordjoined

type LeadingQueryWordChoice string

const (
	RarestFullyListedQueryWord     LeadingQueryWordChoice = "rarest word, fully listed"
	MoreCommonFullyListedQueryWord LeadingQueryWordChoice = "more common word, fully listed"
	RarestPartlyListedQueryWord    LeadingQueryWordChoice = "rarest word, partly listed"
)

func leadingQueryWordChoiceOf(round matchedAndHeldDocumentsRound) LeadingQueryWordChoice {
	leadingQueryWord := round.leadingQueryWord()
	switch {
	case !leadingQueryWord.isFullyListed():
		return RarestPartlyListedQueryWord
	case leadingQueryWord.word == round.queryWordsFewestDocumentsFirst[0].word:
		return RarestFullyListedQueryWord
	default:
		return MoreCommonFullyListedQueryWord
	}
}
