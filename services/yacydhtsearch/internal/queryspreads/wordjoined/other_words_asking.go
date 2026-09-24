package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type OtherWordsAsking string

const (
	OtherWordsAskedForTheCandidates          OtherWordsAsking = "for the candidates"
	OtherWordsNotAsked                       OtherWordsAsking = "no candidates"
	OtherWordsAskedOverTheCeiling            OtherWordsAsking = "whole, candidates over the ceiling"
	OtherWordsAskedBesideAPartialLeadingWord OtherWordsAsking = "whole, partial leading word"
	OtherWordsAskedWithoutASample            OtherWordsAsking = "whole, no sample"
)

func otherWordsAskingFor(
	leadingWordPartition wordPartition,
	candidates []yacymodel.URLHash,
	documentsToMatchCeiling int,
) OtherWordsAsking {
	switch {
	case len(candidates) == 0:
		return OtherWordsNotAsked
	case !leadingWordPartition.hasACompleteAbstract():
		return OtherWordsAskedBesideAPartialLeadingWord
	case len(candidates) > documentsToMatchCeiling:
		return OtherWordsAskedOverTheCeiling
	default:
		return OtherWordsAskedForTheCandidates
	}
}

func (asking OtherWordsAsking) asksAmong(
	asksOfTheOtherWords discoveryAsks,
	candidates []yacymodel.URLHash,
) discoveryAsks {
	switch asking {
	case OtherWordsNotAsked:
		return nil
	case OtherWordsAskedForTheCandidates:
		return asksOfTheOtherWords.matching(candidates)
	default:
		return asksOfTheOtherWords
	}
}
