package wordjoined

import "slices"

type LeadingQueryWordStanding string

const (
	RarestFullyListedQueryWord     LeadingQueryWordStanding = "rarest word, fully listed"
	MoreCommonFullyListedQueryWord LeadingQueryWordStanding = "more common word, fully listed"
	RarestPartlyListedQueryWord    LeadingQueryWordStanding = "rarest word, partly listed"
)

func leadingQueryWordStandingAmong(
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas,
) LeadingQueryWordStanding {
	switch slices.IndexFunc(queryWordsFewestDocumentsFirst, queryWordAcrossReplicas.isFullyListed) {
	case -1:
		return RarestPartlyListedQueryWord
	case 0:
		return RarestFullyListedQueryWord
	default:
		return MoreCommonFullyListedQueryWord
	}
}
