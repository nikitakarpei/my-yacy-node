package replicaasks

import "time"

type PerformedReplicaAsks struct {
	EndedBy        EndedBy
	TimeSpent      time.Duration
	WordPartitions []PerformedWordPartition
}

type PerformedWordPartition struct {
	SettledBy               SettledBy
	CoveringAskPutOn        PutOn
	AmountOfDocumentsListed int
	AsksPutOn               []PutOn
}

type EndedBy string

const (
	EndedByCoverage EndedBy = "coverage"
	EndedByDeadline EndedBy = "deadline"
)

type SettledBy string

const (
	SettledByCoverage      SettledBy = "coverage"
	SettledByNoReplicaLeft SettledBy = "no replica left"
	SettledByDeadline      SettledBy = "deadline"
)

type PutOn string

const (
	PutOnStart       PutOn = "start"
	PutOnHedgeDelay  PutOn = "hedge delay"
	PutOnEmptyAnswer PutOn = "empty answer"
	PutOnFailure     PutOn = "failure"
)
