package searchresult

type Outcome int

const (
	PeersAsked Outcome = iota
	NoIndexedWordInQuery
	NoPeerToAsk
)
