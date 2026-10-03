package documentasks

type DocumentsToMatchDecision string

const (
	NamedTheDocumentsToMatch         DocumentsToMatchDecision = "naming the documents to match"
	NamedNoneOverTheCeiling          DocumentsToMatchDecision = "naming none, over the ceiling"
	NamedNonePredictedOverTheCeiling DocumentsToMatchDecision = "naming none, predicted over the ceiling"
	NoDocumentsToMatch               DocumentsToMatchDecision = "skipped, no documents to match"
)

type DocumentsToMatchDecisionPerPartition map[uint]DocumentsToMatchDecision

func (decisionPerPartition DocumentsToMatchDecisionPerPartition) AmountOfPartitionsPerDecision() map[DocumentsToMatchDecision]int {
	amountOfPartitionsPerDecision := map[DocumentsToMatchDecision]int{}
	for _, decision := range decisionPerPartition {
		amountOfPartitionsPerDecision[decision]++
	}

	return amountOfPartitionsPerDecision
}
