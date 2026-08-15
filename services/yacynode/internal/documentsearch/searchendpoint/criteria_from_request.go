package searchendpoint

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	defaultSearchCount = 10
	defaultSearchTime  = 3 * time.Second
	maxSearchTime      = 3 * time.Second
)

const (
	ftpSitePrefix  = "ftp."
	siteHTTPScheme = "http://"
	siteFTPScheme  = "ftp://"
)

func criteriaFromRequest(req yacyproto.SearchRequest) (searchcriteria.Criteria, error) {
	operators := queryOperatorsIn(req.Modifier)
	siteHash, err := siteHashFromRequest(req, operators)
	if err != nil {
		return searchcriteria.Criteria{}, err
	}
	maxResults := req.Count
	if maxResults <= 0 {
		maxResults = defaultSearchCount
	}
	return searchcriteria.Criteria{
		Terms:              req.Query,
		ExcludedTerms:      req.Exclude,
		RequiredDocuments:  req.URLs,
		MaxResults:         maxResults,
		MaxTermSpread:      req.MaxDist,
		TimeLimit:          timeLimitFromRequest(req),
		ContentKind:        contentKindFromDomain(req.ContentDom),
		StrictContentKind:  req.StrictContentDom,
		RequiredAppearance: req.RequiredAppearance,
		// Deliberate divergence from YaCy: only the /language/ modifier filters; the
		// plain language field drives YaCy's ranking boost, which this node omits.
		Language: operators.Language,
		SiteHash: siteHash,
	}, nil
}

const (
	operatorLanguagePrefix = "/language/"
	operatorSitePrefix     = "site:"
)

type queryOperators struct {
	Language yacymodel.Optional[yacymodel.Language]
	SiteHost string
}

func queryOperatorsIn(modifier string) queryOperators {
	var parsed queryOperators
	for token := range strings.FieldsSeq(modifier) {
		switch {
		case strings.HasPrefix(token, operatorLanguagePrefix):
			code := strings.ToLower(token[len(operatorLanguagePrefix):])
			if language, err := yacymodel.ParseLanguage(code); err == nil {
				parsed.Language = yacymodel.Some(language)
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
	hash, err := hostHashOfSite(siteHost)
	if err != nil {
		return yacymodel.None[yacymodel.HostHash](), fmt.Errorf("site hash: %w", err)
	}

	return yacymodel.Some(hash), nil
}

func hostHashOfSite(site string) (yacymodel.HostHash, error) {
	host := strings.Trim(strings.ToLower(site), ".")
	if host == "" {
		return yacymodel.HostHash{}, fmt.Errorf("site %q names no host", site)
	}

	scheme := siteHTTPScheme
	if strings.HasPrefix(host, ftpSitePrefix) {
		scheme = siteFTPScheme
	}
	address, err := url.Parse(scheme + host)
	if err != nil {
		return yacymodel.HostHash{}, fmt.Errorf("parse site %q: %w", site, err)
	}
	if address.Hostname() == "" {
		return yacymodel.HostHash{}, fmt.Errorf("site %q names no host", site)
	}

	return yacymodel.URLNormalformOf(address).HostHash(), nil
}

func timeLimitFromRequest(req yacyproto.SearchRequest) time.Duration {
	timeLimit := time.Duration(req.Time) * time.Millisecond
	if timeLimit <= 0 {
		return defaultSearchTime
	}

	return min(timeLimit, maxSearchTime)
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
