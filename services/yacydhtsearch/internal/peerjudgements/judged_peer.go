package peerjudgements

type Judgement string

const (
	Honored    Judgement = "honored"
	Ignored    Judgement = "ignored"
	NoEvidence Judgement = "no evidence"
)

type JudgedPeer struct {
	PeerAtVersion
	Judgement Judgement
}
