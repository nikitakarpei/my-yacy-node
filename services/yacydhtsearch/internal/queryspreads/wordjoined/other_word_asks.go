package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type OtherWordAsks string

const (
	OtherWordAsksNamingTheDocumentsToMatch OtherWordAsks = "naming the documents to match"
	OtherWordAsksOverTheCeiling            OtherWordAsks = "naming none, over the ceiling"
	OtherWordAsksSkipped                   OtherWordAsks = "skipped, no documents to match"
	OtherWordAsksPredictedOverTheCeiling   OtherWordAsks = "naming none, predicted over the ceiling"
)

func otherWordAsksFrom(
	documentsToMatch []yacymodel.URLHash,
	documentsToMatchCeiling int,
) OtherWordAsks {
	switch {
	case len(documentsToMatch) == 0:
		return OtherWordAsksSkipped
	case len(documentsToMatch) > documentsToMatchCeiling:
		return OtherWordAsksOverTheCeiling
	default:
		return OtherWordAsksNamingTheDocumentsToMatch
	}
}

func predictsTheOtherWordsOverTheCeiling(
	rememberedAmountOfTheLeadingWord int,
	partitions yacymodel.DHTRingPartitions,
	documentsToMatchCeiling int,
) bool {
	return rememberedAmountOfTheLeadingWord/int(partitions) > documentsToMatchCeiling
}

func (otherWordAsks OtherWordAsks) appliedTo(
	asksOfTheOtherWords discoveryAsks,
	documentsToMatch []yacymodel.URLHash,
) discoveryAsks {
	switch otherWordAsks {
	case OtherWordAsksNamingTheDocumentsToMatch:
		return asksOfTheOtherWords.forDocumentsToMatch(documentsToMatch)
	case OtherWordAsksOverTheCeiling, OtherWordAsksPredictedOverTheCeiling:
		return asksOfTheOtherWords
	case OtherWordAsksSkipped:
	}

	return nil
}
