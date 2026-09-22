package peerjudgements

import "time"

type RecordedJudgement struct {
	Question Question
	JudgedPeer
	JudgedAt time.Time
}
