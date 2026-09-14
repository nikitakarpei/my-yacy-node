package peerasks

type AskedFor string

const (
	MatchedItems  AskedFor = "matched items"
	HeldDocuments AskedFor = "held documents"
	URLMetadata   AskedFor = "url metadata"
)
