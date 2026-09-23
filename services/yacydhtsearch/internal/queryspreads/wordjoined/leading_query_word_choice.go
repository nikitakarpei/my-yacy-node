package wordjoined

type LeadingQueryWordChoice string

const (
	RarestQueryWordWithCompleteAbstracts     LeadingQueryWordChoice = "rarest word, complete abstracts"
	MoreCommonQueryWordWithCompleteAbstracts LeadingQueryWordChoice = "more common word, complete abstracts"
	RarestQueryWordWithoutCompleteAbstracts  LeadingQueryWordChoice = "rarest word, partial abstracts"
)

func leadingQueryWordChoiceOf(round abstractsRound) LeadingQueryWordChoice {
	leadingQueryWord := round.leadingQueryWord()
	switch {
	case !leadingQueryWord.hasCompleteAbstracts():
		return RarestQueryWordWithoutCompleteAbstracts
	case leadingQueryWord.word == round.queryWordsFewestDocumentsFirst[0].word:
		return RarestQueryWordWithCompleteAbstracts
	default:
		return MoreCommonQueryWordWithCompleteAbstracts
	}
}
