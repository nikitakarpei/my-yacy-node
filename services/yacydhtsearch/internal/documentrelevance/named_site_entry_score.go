package documentrelevance

import (
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	prefixOfWorldWideWebHost = "www."

	namedSiteEntryScoreOfMalformedAddress        = 0.0
	namedSiteEntryScoreOfSiteTheQueryDoesNotName = 0.0

	amountOfStepsToSiteEntry = 1
)

func namedSiteEntryScoreOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) float64 {
	address, err := url.Parse(foundDocument.Address)
	if err != nil {
		return namedSiteEntryScoreOfMalformedAddress
	}
	wordsInSiteName := wordsInSiteNameOf(address.Hostname())
	amountOfQueryWordsInSiteName := amountOfQueryWordsAmong(wordsInSiteName, queryWords)
	if amountOfQueryWordsInSiteName == 0 {
		return namedSiteEntryScoreOfSiteTheQueryDoesNotName
	}

	shareOfSiteNameTheQueryNames := float64(amountOfQueryWordsInSiteName) /
		float64(len(wordsInSiteName))
	shareOfQueryTheSiteNameHolds := float64(amountOfQueryWordsInSiteName) /
		float64(len(queryWords))
	amountOfStepsToDocument := amountOfStepsToSiteEntry +
		amountOfPathSegmentsOf(address.Path)

	return shareOfSiteNameTheQueryNames * shareOfQueryTheSiteNameHolds /
		float64(amountOfStepsToDocument)
}

func wordsInSiteNameOf(host string) map[yacymodel.Hash]struct{} {
	siteName := strings.TrimPrefix(host, prefixOfWorldWideWebHost)
	if placeOfLastDot := strings.LastIndex(siteName, "."); placeOfLastDot >= 0 {
		siteName = siteName[:placeOfLastDot]
	}

	return wordsIn(siteName)
}

func amountOfQueryWordsAmong(
	words map[yacymodel.Hash]struct{}, queryWords []yacymodel.Hash,
) int {
	amountOfQueryWords := 0
	for _, queryWord := range queryWords {
		if _, amongWords := words[queryWord]; !amongWords {
			continue
		}
		amountOfQueryWords++
	}

	return amountOfQueryWords
}

func amountOfPathSegmentsOf(path string) int {
	amountOfPathSegments := 0
	for segment := range strings.SplitSeq(path, "/") {
		if segment == "" {
			continue
		}
		amountOfPathSegments++
	}

	return amountOfPathSegments
}
