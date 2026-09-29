package spamfeatures

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagesite"
)

const pathWordReadBytes = 30

var (
	redirectPathPattern = regexp.MustCompile(
		`(?i)^/(go|out|play|visit|recommends?|links?|redirect|refer|click|track|aff|goto|jump|` +
			`get|offer|bonus)(/|$)`,
	)
	affiliateQueryKeys = map[string]bool{
		"aff": true, "affid": true, "aff_id": true, "affiliate": true, "affiliate_id": true,
		"btag": true, "clickid": true, "click_id": true, "subid": true, "sub_id": true,
		"tracking": true, "partner": true, "ref": true, "refid": true, "ref_id": true,
		"tag": true, "cid": true, "pid": true,
	}
)

type linkReading struct {
	words                       []string
	amountOfLocalLinks          int
	amountOfExternalLinks       int
	amountOfRedirectPathLinks   int
	amountOfAffiliateQueryLinks int
	topOutboundDomainShare      float64
	outboundLinksOfDomain       map[string]int
}

func linkReadingOf(page htmlreading.Reading) linkReading {
	pageDomain := pagesite.RegistrableDomainOf(page.Address)
	reading := linkReading{outboundLinksOfDomain: map[string]int{}}
	for _, link := range page.WebLinks {
		linkDomain := pagesite.RegistrableDomainOf(link)
		if linkDomain == pageDomain {
			reading.amountOfLocalLinks++
		}
		reading.readLink(link, linkDomain, pageDomain)
	}
	reading.amountOfExternalLinks = len(page.WebLinks) - reading.amountOfLocalLinks
	for _, dataLink := range page.DataLinks {
		reading.readLink(
			dataLink.Address,
			pagesite.RegistrableDomainOf(dataLink.Address),
			pageDomain,
		)
	}
	reading.topOutboundDomainShare = topShareOf(reading.outboundLinksOfDomain)
	return reading
}

func (r *linkReading) readLink(link canonicalurl.CanonicalURL, linkDomain, pageDomain string) {
	if linkDomain == pageDomain {
		r.readLocal(link.WebAddress())
		return
	}
	r.readOutbound(link, linkDomain)
}

func (r *linkReading) readLocal(webAddress *url.URL) {
	if !redirectPathPattern.MatchString(webAddress.EscapedPath()) {
		return
	}
	r.amountOfRedirectPathLinks++
	r.words = append(r.words, "localpath:"+pathWordOf(webAddress.EscapedPath()))
}

func (r *linkReading) readOutbound(link canonicalurl.CanonicalURL, linkDomain string) {
	webAddress := link.WebAddress()
	r.outboundLinksOfDomain[linkDomain]++
	r.words = append(r.words,
		"domain:"+linkDomain,
		"host:"+link.Hostname(),
		"domainpath:"+linkDomain+"/"+pathWordOf(webAddress.EscapedPath()),
	)
	if hasAffiliateQueryKey(webAddress.Query()) {
		r.amountOfAffiliateQueryLinks++
	}
}

func pathWordOf(escapedPath string) string {
	firstSegment, _, _ := strings.Cut(strings.Trim(escapedPath, "/"), "/")
	return strings.ToLower(firstSegment[:min(len(firstSegment), pathWordReadBytes)])
}

func hasAffiliateQueryKey(query url.Values) bool {
	for key := range query {
		if affiliateQueryKeys[strings.ToLower(key)] {
			return true
		}
	}
	return false
}

func topShareOf(outboundLinksOfDomain map[string]int) float64 {
	outboundLinks, topLinks := 0, 0
	for _, links := range outboundLinksOfDomain {
		outboundLinks += links
		topLinks = max(topLinks, links)
	}
	if outboundLinks == 0 {
		return 0
	}
	return float64(topLinks) / float64(outboundLinks)
}
