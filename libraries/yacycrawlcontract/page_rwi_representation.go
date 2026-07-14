package yacycrawlcontract

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWIRepresentation struct {
	CanonicalURL string
	Metadata     []yacymodel.URIMetadataRow
	Postings     []yacymodel.RWIPosting
}
