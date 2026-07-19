package yacycrawlcontract

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWIRepresentation struct {
	CanonicalURL string                  `json:"CanonicalURL"`
	Metadata     []yacymodel.URLMetadata `json:"Metadata"`
	Postings     []yacymodel.RWIPosting  `json:"Postings"`
}
