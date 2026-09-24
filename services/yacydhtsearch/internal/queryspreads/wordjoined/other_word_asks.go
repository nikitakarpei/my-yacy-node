package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type OtherWordAsks string

const (
	OtherWordAsksNamingTheDocumentsToMatch OtherWordAsks = "naming the documents to match"
	OtherWordAsksOverTheCeiling            OtherWordAsks = "naming none, over the ceiling"
	OtherWordAsksSkipped                   OtherWordAsks = "skipped, no documents to match"
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

func (otherWordAsks OtherWordAsks) appliedTo(
	asksOfTheOtherWords discoveryAsks,
	documentsToMatch []yacymodel.URLHash,
) discoveryAsks {
	switch otherWordAsks {
	case OtherWordAsksNamingTheDocumentsToMatch:
		return asksOfTheOtherWords.forDocumentsToMatch(documentsToMatch)
	case OtherWordAsksOverTheCeiling:
		return asksOfTheOtherWords
	case OtherWordAsksSkipped:
	}

	return nil
}
