package yacycrawlcontract

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWIRepresentation struct {
	CanonicalURL string                         `json:"CanonicalURL"`
	Metadata     []yacymodel.URIMetadataRow     `json:"Metadata"`
	Postings     []yacymodel.RWIPostingWireForm `json:"Postings"`
}
