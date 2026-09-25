package replicaasks

type Coverage struct {
	AmountOfDocumentsListed int
	PeerSearched            bool
}

func (coverage Coverage) coversThePartition() bool {
	return coverage.AmountOfDocumentsListed > 0 || coverage.PeerSearched
}
