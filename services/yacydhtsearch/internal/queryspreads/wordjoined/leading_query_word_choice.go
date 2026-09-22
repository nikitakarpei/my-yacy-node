package wordjoined

import "slices"

type LeadingQueryWordChoice string

const (
	RarestFullyListedQueryWord     LeadingQueryWordChoice = "rarest word, fully listed"
	MoreCommonFullyListedQueryWord LeadingQueryWordChoice = "more common word, fully listed"
	RarestPartlyListedQueryWord    LeadingQueryWordChoice = "rarest word, partly listed"
)

func leadingQueryWordChoiceAmong(
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas,
) LeadingQueryWordChoice {
	switch slices.IndexFunc(queryWordsFewestDocumentsFirst, queryWordAcrossReplicas.isFullyListed) {
	case -1:
		return RarestPartlyListedQueryWord
	case 0:
		return RarestFullyListedQueryWord
	default:
		return MoreCommonFullyListedQueryWord
	}
}
