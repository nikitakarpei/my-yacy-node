package documentsearch

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

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
	contentKind        contentKind
	strictContentKind  bool
	requiredAppearance yacymodel.Optional[yacymodel.Appearance]
	language           yacymodel.Optional[yacymodel.Language]
	siteHash           yacymodel.Optional[yacymodel.HostHash]
}

func searchCriteriaFromRequest(req yacyproto.SearchRequest) (searchCriteria, error) {
	operators := queryOperatorsIn(req.Modifier)
	siteHash, err := siteHashFromRequest(req, operators)
	if err != nil {
		return searchCriteria{}, err
	}
	language, err := languageFromOperators(operators)
	if err != nil {
		return searchCriteria{}, err
	}
	maxResults := req.Count
	if maxResults <= 0 {
		maxResults = defaultSearchCount
	}
	timeLimit := time.Duration(req.Time) * time.Millisecond
	if timeLimit <= 0 {
		timeLimit = defaultSearchTime
	}
	return searchCriteria{
		terms:              req.Query,
		excludedTerms:      req.Exclude,
		requiredDocuments:  req.URLs,
		maxResults:         maxResults,
		maxTermSpread:      req.MaxDist,
		timeLimit:          timeLimit,
		contentKind:        contentKindFromDomain(req.ContentDom),
		strictContentKind:  req.StrictContentDom,
		requiredAppearance: req.RequiredAppearance,
		// Deliberate divergence from YaCy: only the /language/ modifier filters; the
		// plain language field drives YaCy's ranking boost, which this node omits.
		language: language,
		siteHash: siteHash,
	}, nil
}

const (
	operatorLanguagePrefix = "/language/"
	operatorSitePrefix     = "site:"
	operatorLanguageLength = 2
)

type queryOperators struct {
	Language string
	SiteHost string
}

func queryOperatorsIn(modifier string) queryOperators {
	var parsed queryOperators
	for token := range strings.FieldsSeq(modifier) {
		switch {
		case strings.HasPrefix(token, operatorLanguagePrefix):
			if code := token[len(operatorLanguagePrefix):]; len(code) == operatorLanguageLength {
				parsed.Language = strings.ToLower(code)
			}
		case strings.HasPrefix(token, operatorSitePrefix):
			parsed.SiteHost = token[len(operatorSitePrefix):]
		}
	}

	return parsed
}

func siteHashFromRequest(
	req yacyproto.SearchRequest, operators queryOperators,
) (yacymodel.Optional[yacymodel.HostHash], error) {
	if req.SiteHash != "" {
		hash, err := yacymodel.ParseHostHash(req.SiteHash)
		if err != nil {
			return yacymodel.None[yacymodel.HostHash](), fmt.Errorf("site hash: %w", err)
		}

		return yacymodel.Some(hash), nil
	}
	siteHost := operators.SiteHost
	if siteHost == "" {
		siteHost = req.SiteHost
	}
	if siteHost == "" {
		return yacymodel.None[yacymodel.HostHash](), nil
	}
	hash, err := yacymodel.HashHost(siteHost)
	if err != nil {
		return yacymodel.None[yacymodel.HostHash](), fmt.Errorf("site hash: %w", err)
	}

	return yacymodel.Some(hash), nil
}

func languageFromOperators(
	operators queryOperators,
) (yacymodel.Optional[yacymodel.Language], error) {
	if operators.Language == "" {
		return yacymodel.None[yacymodel.Language](), nil
	}
	language, err := yacymodel.ParseLanguage(operators.Language)
	if err != nil {
		return yacymodel.None[yacymodel.Language](), fmt.Errorf("language: %w", err)
	}

	return yacymodel.Some(language), nil
}

func contentKindFromDomain(domain yacyproto.SearchContentDomain) contentKind {
	switch domain {
	case yacyproto.ContentDomainImage:
		return imageContent
	case yacyproto.ContentDomainAudio:
		return audioContent
	case yacyproto.ContentDomainVideo:
		return videoContent
	case yacyproto.ContentDomainApp:
		return applicationContent
	default:
		return anyContent
	}
}
