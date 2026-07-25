// Package profileadmission decides which URLs a crawl profile lets into a crawl order.
package profileadmission

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Admission struct {
	scope           yacycrawlcontract.CrawlScope
	maxDepth        int
	allowQueryURLs  bool
	maxPagesPerHost int
	maxAdmittedURLs int
	urlMustMatch    *regexp.Regexp
	urlMustNotMatch *regexp.Regexp
	seedHosts       map[string]struct{}
	seedDirectories []string
	admittedURLs    map[string]struct{}
	pagesPerHost    map[string]int
}

func New(
	profile yacycrawlcontract.CrawlProfile,
	canonicalSeeds []string,
	maxAdmittedURLs int,
) (*Admission, error) {
	mustMatch, err := regexp.Compile(matchOrAll(profile.URLMustMatch))
	if err != nil {
		return nil, fmt.Errorf("compile url must match: %w", err)
	}
	var mustNotMatch *regexp.Regexp
	if profile.URLMustNotMatch != "" {
		mustNotMatch, err = regexp.Compile(profile.URLMustNotMatch)
		if err != nil {
			return nil, fmt.Errorf("compile url must not match: %w", err)
		}
	}

	admission := &Admission{
		scope:           profile.Scope,
		maxDepth:        profile.MaxDepth,
		allowQueryURLs:  profile.AllowQueryURLs,
		maxPagesPerHost: profile.MaxPagesPerHost,
		maxAdmittedURLs: maxAdmittedURLs,
		urlMustMatch:    mustMatch,
		urlMustNotMatch: mustNotMatch,
		seedHosts:       map[string]struct{}{},
		admittedURLs:    map[string]struct{}{},
		pagesPerHost:    map[string]int{},
	}
	for _, seed := range canonicalSeeds {
		host, directory, err := hostAndDirectory(seed)
		if err != nil {
			return nil, err
		}
		admission.seedHosts[host] = struct{}{}
		admission.seedDirectories = append(admission.seedDirectories, directory)
	}
	return admission, nil
}

func (a *Admission) Admit(canonicalURL string, depth int) bool {
	if depth > a.maxDepth {
		return false
	}
	if _, already := a.admittedURLs[canonicalURL]; already {
		return false
	}
	if len(a.admittedURLs) >= a.maxAdmittedURLs {
		return false
	}
	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return false
	}
	if parsed.RawQuery != "" && !a.allowQueryURLs {
		return false
	}
	if !a.urlMustMatch.MatchString(canonicalURL) {
		return false
	}
	if a.urlMustNotMatch != nil && a.urlMustNotMatch.MatchString(canonicalURL) {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if !a.withinScope(host, canonicalURL) {
		return false
	}
	if a.maxPagesPerHost != yacycrawlcontract.UnlimitedPagesPerHost &&
		a.pagesPerHost[host] >= a.maxPagesPerHost {
		return false
	}

	a.admittedURLs[canonicalURL] = struct{}{}
	a.pagesPerHost[host]++
	return true
}

func (a *Admission) withinScope(host, canonicalURL string) bool {
	switch a.scope {
	case yacycrawlcontract.ScopeWide:
		return true
	case yacycrawlcontract.ScopeDomain:
		_, ok := a.seedHosts[host]
		return ok
	case yacycrawlcontract.ScopeSubpath:
		for _, directory := range a.seedDirectories {
			if strings.HasPrefix(canonicalURL, directory) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func hostAndDirectory(canonicalURL string) (host, directory string, err error) {
	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return "", "", fmt.Errorf("parse seed url: %w", err)
	}
	host = strings.ToLower(parsed.Hostname())
	trimmed := canonicalURL
	if slash := strings.LastIndexByte(trimmed, '/'); slash >= 0 {
		trimmed = trimmed[:slash+1]
	}
	return host, trimmed, nil
}

func matchOrAll(pattern string) string {
	if pattern == "" {
		return yacycrawlcontract.MatchAll
	}
	return pattern
}
