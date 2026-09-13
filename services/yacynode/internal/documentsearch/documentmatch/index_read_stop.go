package documentmatch

type IndexReadStop int

const (
	IndexReadStoppedAtEndOfWord IndexReadStop = iota
	IndexReadStoppedAtRelevanceBound
	IndexReadStoppedAtDeadline
)
