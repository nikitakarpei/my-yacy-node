package peerjudgements

import "time"

type RecordedJudgement struct {
	Form Form
	JudgedPeer
	JudgedAt time.Time
}
