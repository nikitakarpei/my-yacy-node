package spamfeatures

import (
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagelinks"
)

const (
	hostLengthScale       = 20
	addressDepthScale     = 5
	hiddenDisplayScale    = 5
	anchorScale           = 100
	scriptScale           = 20
	hostWordShortestRunes = 4
	wwwPrefix             = "www."
)

var (
	sponsoredPattern = regexp.MustCompile(`rel\s*=\s*["'][^"']*sponsored`)
	hostWordsIgnored = map[string]bool{"html": true, "blogspot": true, "wordpress": true}
)

func metaValuesOf(page htmlreading.Reading, links linkReading, lowercaseMarkup string) []float64 {
	host := page.Address.Hostname()
	return []float64{
		logOnePlus(utf8.RuneCountInString(page.VisibleText)),
		logOnePlus(page.BodyBytes),
		logOnePlus(links.amountOfLocalLinks),
		logOnePlus(links.amountOfExternalLinks),
		float64(links.amountOfExternalLinks) /
			float64(links.amountOfLocalLinks+links.amountOfExternalLinks+1),
		float64(strings.Count(host, "-")),
		float64(amountOfDigits(host)),
		float64(len(host)) / hostLengthScale,
		float64(strings.Count(host, ".")),
		oneWhen(page.Address.HasQuery()),
		float64(strings.Count(page.Address.String(), "/")) / addressDepthScale,
		oneWhen(textNamesHost(page.VisibleText, host)),
		float64(strings.Count(lowercaseMarkup, "display:none")) / hiddenDisplayScale,
		float64(strings.Count(lowercaseMarkup, "<a ")) / anchorScale,
		float64(strings.Count(lowercaseMarkup, "<script")) / scriptScale,
		logOnePlus(len(sponsoredPattern.FindAllStringIndex(lowercaseMarkup, -1))),
		logOnePlus(links.amountOfRedirectPathLinks),
		logOnePlus(links.amountOfAffiliateQueryLinks),
		logOnePlus(len(page.DataLinks)),
		logOnePlus(amountOfEncodedLinks(page.DataLinks)),
		links.topOutboundDomainShare,
	}
}

func logOnePlus(count int) float64 {
	return math.Log1p(float64(count))
}

func oneWhen(condition bool) float64 {
	if condition {
		return 1
	}
	return 0
}

func amountOfDigits(host string) int {
	digits := 0
	for _, character := range host {
		if unicode.IsDigit(character) {
			digits++
		}
	}
	return digits
}

func textNamesHost(visibleText, host string) bool {
	hostWords := hostWordsOf(host)
	if len(hostWords) == 0 {
		return true
	}
	lowercaseText := strings.ToLower(visibleText)
	for _, word := range hostWords {
		if strings.Contains(lowercaseText, word) {
			return true
		}
	}
	return false
}

func hostWordsOf(host string) []string {
	var hostWords []string
	for _, word := range strings.FieldsFunc(strings.ReplaceAll(host, wwwPrefix, ""), isNotLetter) {
		if len(word) >= hostWordShortestRunes && !hostWordsIgnored[word] {
			hostWords = append(hostWords, word)
		}
	}
	return hostWords
}

func isNotLetter(character rune) bool {
	return character < 'a' || character > 'z'
}

func amountOfEncodedLinks(dataLinks []pagelinks.DataLink) int {
	encodedLinks := 0
	for _, dataLink := range dataLinks {
		if dataLink.Encoded {
			encodedLinks++
		}
	}
	return encodedLinks
}
