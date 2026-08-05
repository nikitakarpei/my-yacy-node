package searchendpoint

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	defaultSearchCount = 10
	defaultSearchTime  = 3 * time.Second
)

func criteriaFromRequest(req yacyproto.SearchRequest) (searchcriteria.Criteria, error) {
	operators := queryOperatorsIn(req.Modifier)
	siteHash, err := siteHashFromRequest(req, operators)
	if err != nil {
		return searchcriteria.Criteria{}, err
	}
	language, err := languageFromOperators(operators)
	if err != nil {
		return searchcriteria.Criteria{}, err
	}
	maxResults := req.Count
	if maxResults <= 0 {
		maxResults = defaultSearchCount
	}
	timeLimit := time.Duration(req.Time) * time.Millisecond
	if timeLimit <= 0 {
		timeLimit = defaultSearchTime
	}

	return searchcriteria.Criteria{
		Terms:              req.Query,
		ExcludedTerms:      req.Exclude,
		RequiredDocuments:  req.URLs,
		MaxResults:         maxResults,
		MaxTermSpread:      req.MaxDist,
		TimeLimit:          timeLimit,
		ContentKind:        contentKindFromDomain(req.ContentDom),
		StrictContentKind:  req.StrictContentDom,
		RequiredAppearance: req.RequiredAppearance,
		// Deliberate divergence from YaCy: only the /language/ modifier filters; the
		// plain language field drives YaCy's ranking boost, which this node omits.
		Language: language,
		SiteHash: siteHash,
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

func contentKindFromDomain(domain yacyproto.SearchContentDomain) searchcriteria.ContentKind {
	switch domain {
	case yacyproto.ContentDomainImage:
		return searchcriteria.ImageContent
	case yacyproto.ContentDomainAudio:
		return searchcriteria.AudioContent
	case yacyproto.ContentDomainVideo:
		return searchcriteria.VideoContent
	case yacyproto.ContentDomainApp:
		return searchcriteria.ApplicationContent
	default:
		return searchcriteria.AnyContent
	}
}
