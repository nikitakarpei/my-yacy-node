package matchingwords

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type AsksPerPartition map[uint]Kind

type Kind string

const (
	NamingTheDocumentsToMatch Kind = "naming the documents to match"
	OverTheCeiling            Kind = "naming none, over the ceiling"
	Skipped                   Kind = "skipped, no documents to match"
	PredictedOverTheCeiling   Kind = "naming none, predicted over the ceiling"
)

func kindFrom(documentsToMatch []yacymodel.URLHash, documentsToMatchCeiling int) Kind {
	switch {
	case len(documentsToMatch) == 0:
		return Skipped
	case len(documentsToMatch) > documentsToMatchCeiling:
		return OverTheCeiling
	default:
		return NamingTheDocumentsToMatch
	}
}

func (kind Kind) askIn(
	partition uint,
	matchingWords []yacymodel.Hash,
	documentsToMatch []yacymodel.URLHash,
	run Run,
) {
	switch kind {
	case NamingTheDocumentsToMatch:
		run.AskPartitionForWordsAmong(partition, matchingWords, documentsToMatch)
	case OverTheCeiling, PredictedOverTheCeiling:
		run.AskPartitionFor(partition, matchingWords)
	case Skipped:
	}
}
