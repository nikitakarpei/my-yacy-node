package replicaasks

import "time"

type PerformedReplicaAsks struct {
	EndedBy        EndedBy
	TimeSpent      time.Duration
	WordPartitions []PerformedWordPartition
}

type PerformedWordPartition struct {
	SettledBy               SettledBy
	AmountOfDocumentsListed int
	Asks                    []PerformedReplicaAsk
}

type PerformedReplicaAsk struct {
	PutAs PutAs
}

type EndedBy string

const (
	EndedByCoverage EndedBy = "coverage"
	EndedByDeadline EndedBy = "deadline"
)

type SettledBy string

const (
	SettledByFirst              SettledBy = "first"
	SettledByHedge              SettledBy = "hedge"
	SettledByAfterAnEmptyAnswer SettledBy = "after empty answer"
	SettledByAfterAFailure      SettledBy = "after failure"
	SettledByExhausted          SettledBy = "exhausted"
	SettledByDeadline           SettledBy = "deadline"
)

type PutAs string

const (
	PutAsFirst              PutAs = "first"
	PutAsHedge              PutAs = "hedge"
	PutAsAfterAnEmptyAnswer PutAs = "after empty answer"
	PutAsAfterAFailure      PutAs = "after failure"
)

func settledByOf(putAs PutAs) SettledBy {
	return SettledBy(putAs)
}
