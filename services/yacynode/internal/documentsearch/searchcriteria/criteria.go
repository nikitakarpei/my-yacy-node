// Package searchcriteria holds the terms, filters and limits that one search
// request asks the index for.
package searchcriteria

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Criteria struct {
	Terms              []yacymodel.Hash
	ExcludedTerms      []yacymodel.Hash
	RequiredDocuments  []yacymodel.URLHash
	MaxResults         int
	MaxTermSpread      int
	TimeLimit          time.Duration
	ContentKind        ContentKind
	StrictContentKind  bool
	RequiredAppearance yacymodel.Optional[yacymodel.Appearance]
	Language           yacymodel.Optional[yacymodel.Language]
	SiteHash           yacymodel.Optional[yacymodel.HostHash]
}
