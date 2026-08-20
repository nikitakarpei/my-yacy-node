// Package profileadmission decides which URLs a crawl profile lets into a crawl order.
package profileadmission

import (
	"fmt"
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
	admittedURLs    map[yacycrawlcontract.CanonicalURL]struct{}
	pagesPerHost    map[string]int
}

func New(
	profile yacycrawlcontract.CrawlProfile,
	canonicalSeeds []yacycrawlcontract.CanonicalURL,
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
		admittedURLs:    map[yacycrawlcontract.CanonicalURL]struct{}{},
		pagesPerHost:    map[string]int{},
	}
	for _, seed := range canonicalSeeds {
		host, directory := hostAndDirectory(seed)
		admission.seedHosts[host] = struct{}{}
		admission.seedDirectories = append(admission.seedDirectories, directory)
	}
	return admission, nil
}

func (a *Admission) Admit(canonicalURL yacycrawlcontract.CanonicalURL, depth int) bool {
	if depth > a.maxDepth {
		return false
	}
	if _, already := a.admittedURLs[canonicalURL]; already {
		return false
	}
	if len(a.admittedURLs) >= a.maxAdmittedURLs {
		return false
	}
	if canonicalURL.HasQuery() && !a.allowQueryURLs {
		return false
	}
	if !a.urlMustMatch.MatchString(canonicalURL.String()) {
		return false
	}
	if a.urlMustNotMatch != nil && a.urlMustNotMatch.MatchString(canonicalURL.String()) {
		return false
	}
	host := canonicalURL.Hostname()
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

func (a *Admission) withinScope(host string, canonicalURL yacycrawlcontract.CanonicalURL) bool {
	switch a.scope {
	case yacycrawlcontract.ScopeWide:
		return true
	case yacycrawlcontract.ScopeDomain:
		_, ok := a.seedHosts[host]
		return ok
	case yacycrawlcontract.ScopeSubpath:
		for _, directory := range a.seedDirectories {
			if strings.HasPrefix(canonicalURL.String(), directory) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func hostAndDirectory(
	canonicalURL yacycrawlcontract.CanonicalURL,
) (host, directory string) {
	directory = canonicalURL.String()
	if slash := strings.LastIndexByte(directory, '/'); slash >= 0 {
		directory = directory[:slash+1]
	}
	return canonicalURL.Hostname(), directory
}

func matchOrAll(pattern string) string {
	if pattern == "" {
		return yacycrawlcontract.MatchAll
	}
	return pattern
}
