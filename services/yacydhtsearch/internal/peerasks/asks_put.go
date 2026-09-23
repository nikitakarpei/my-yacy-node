package peerasks

type AsksPut[Ask any, Answered any] struct {
	Asks         []Ask
	AnsweredAsks []Answered
}
