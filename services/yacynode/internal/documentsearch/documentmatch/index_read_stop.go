package documentmatch

type IndexReadStop int

const (
	StoppedAtEveryPosting IndexReadStop = iota
	StoppedAtRelevanceBound
	StoppedAtDeadline
)
