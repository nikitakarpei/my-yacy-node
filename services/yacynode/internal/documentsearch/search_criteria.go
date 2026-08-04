package documentsearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type reportingMode int

const (
	reportNoMatches reportingMode = iota
	reportTermWithMostMatches
	reportRequestedTerms
)

type requestedMatchReport struct {
	mode  reportingMode
	terms []yacymodel.Hash
}

type contentKind int

const (
	anyContent contentKind = iota
	imageContent
	audioContent
	videoContent
	applicationContent
)

type searchCriteria struct {
	terms              []yacymodel.Hash
	excludedTerms      []yacymodel.Hash
	requiredDocuments  []yacymodel.URLHash
	maxResults         int
	maxTermSpread      int
	timeLimit          time.Duration
	requestedReport    requestedMatchReport
	contentKind        contentKind
	strictContentKind  bool
	requiredAppearance yacymodel.Optional[yacymodel.Appearance]
	language           yacymodel.Optional[yacymodel.Language]
	siteHash           yacymodel.Optional[yacymodel.HostHash]
}
