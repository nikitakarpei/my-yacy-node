package replicaasks

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PerformedReplicaAsks struct {
	AskedFor       peerasks.AskedFor
	EndedBy        EndedBy
	TimeSpent      time.Duration
	WordPartitions []SettledWordPartition
}

type SettledWordPartition struct {
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
	PutOnStart             PutOn = "start"
	PutOnHedgeDelay        PutOn = "hedge delay"
	PutOnNonCoveringAnswer PutOn = "non-covering answer"
	PutOnFailure           PutOn = "failure"
)
