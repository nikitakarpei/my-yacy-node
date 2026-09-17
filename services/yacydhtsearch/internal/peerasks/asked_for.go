package peerasks

type AskedFor string

const (
	MatchedItems            AskedFor = "matched items"
	MatchedAndHeldDocuments AskedFor = "matched and held documents"
	HeldDocuments           AskedFor = "held documents"
	URLMetadata             AskedFor = "url metadata"
)
