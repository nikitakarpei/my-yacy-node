package peerasks

type AskedFor string

const (
	MatchedDocuments        AskedFor = "matched documents"
	MatchedAndHeldDocuments AskedFor = "matched and held documents"
	HeldDocuments           AskedFor = "held documents"
	URLMetadata             AskedFor = "url metadata"
)
