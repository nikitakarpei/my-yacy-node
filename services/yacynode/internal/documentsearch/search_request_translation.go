package documentsearch

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	operatorLanguagePrefix = "/language/"
	operatorSitePrefix     = "site:"
	operatorLanguageLength = 2
)

type queryOperators struct {
	Language string
	SiteHost string
}

func parseQueryOperators(query string) queryOperators {
	var parsed queryOperators
	for token := range strings.FieldsSeq(query) {
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

func searchCriteriaFromRequest(req yacyproto.SearchRequest) (searchCriteria, error) {
	operators := parseQueryOperators(req.Modifier)
	siteHash, err := resolveSiteHash(req, operators)
	if err != nil {
		return searchCriteria{}, err
	}
	language, err := resolveLanguage(operators)
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
		reporting:          matchReportingFromRequest(req),
		contentKind:        contentKindFromDomain(req.ContentDom),
		strictContentKind:  req.StrictContentDom,
		requiredProperties: req.RequiredAppearance,
		// Deliberate divergence from YaCy: only the /language/ modifier filters; the
		// plain language field drives YaCy's ranking boost, which this node omits.
		language: language,
		siteHash: siteHash,
	}, nil
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

func resolveSiteHash(
	req yacyproto.SearchRequest, operators queryOperators,
) (yacymodel.Optional[yacymodel.HostHash], error) {
	if req.SiteHash != "" {
		hash, err := yacymodel.ParseHostHash(req.SiteHash)
		if err != nil {
			return yacymodel.None[yacymodel.HostHash](), fmt.Errorf("site hash: %w", err)
		}

		return yacymodel.Some(hash), nil
	}
	host := firstNonEmpty(operators.SiteHost, req.SiteHost)
	if host == "" {
		return yacymodel.None[yacymodel.HostHash](), nil
	}
	hash, err := yacymodel.HashHost(host)
	if err != nil {
		return yacymodel.None[yacymodel.HostHash](), fmt.Errorf("site hash: %w", err)
	}

	return yacymodel.Some(hash), nil
}

func resolveLanguage(
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

func matchReportingFromRequest(req yacyproto.SearchRequest) matchReporting {
	switch req.Abstracts {
	case "":
		return matchReporting{mode: reportNoMatches}
	case yacyproto.SearchAbstractsAuto:
		return matchReporting{mode: reportTermWithMostMatches}
	default:
		return matchReporting{mode: reportRequestedTerms, terms: req.Abstracts.Hashes()}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
