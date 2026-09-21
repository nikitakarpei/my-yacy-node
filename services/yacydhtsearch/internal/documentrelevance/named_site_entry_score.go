package documentrelevance

import (
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	prefixOfAWorldWideWebHost = "www."

	namedSiteEntryScoreOfAnAddressNoOneCanRead    = 0.0
	namedSiteEntryScoreOfASiteTheQueryDoesNotName = 0.0

	stepsFromTheSiteEntryToItself = 1
)

func namedSiteEntryScoreOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) float64 {
	address, err := url.Parse(foundDocument.Address)
	if err != nil {
		return namedSiteEntryScoreOfAnAddressNoOneCanRead
	}
	wordsInTheSiteName := wordsInTheSiteNameOf(address.Hostname())
	amountOfQueryWordsInTheSiteName := amountOfQueryWordsAmong(wordsInTheSiteName, queryWords)
	if amountOfQueryWordsInTheSiteName == 0 {
		return namedSiteEntryScoreOfASiteTheQueryDoesNotName
	}

	shareOfTheSiteNameTheQueryNames := float64(amountOfQueryWordsInTheSiteName) /
		float64(len(wordsInTheSiteName))
	shareOfTheQueryTheSiteNameHolds := float64(amountOfQueryWordsInTheSiteName) /
		float64(len(queryWords))
	stepsFromTheSiteEntry := stepsFromTheSiteEntryToItself +
		amountOfPathSegmentsOf(address.Path)

	return shareOfTheSiteNameTheQueryNames * shareOfTheQueryTheSiteNameHolds /
		float64(stepsFromTheSiteEntry)
}

func wordsInTheSiteNameOf(host string) map[yacymodel.Hash]struct{} {
	siteName := strings.TrimPrefix(host, prefixOfAWorldWideWebHost)
	if placeOfTheLastDot := strings.LastIndex(siteName, "."); placeOfTheLastDot >= 0 {
		siteName = siteName[:placeOfTheLastDot]
	}
	spelledWords := yacymodel.WordsIn(siteName)

	wordsInTheSiteName := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		wordsInTheSiteName[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return wordsInTheSiteName
}

func amountOfQueryWordsAmong(
	words map[yacymodel.Hash]struct{}, queryWords []yacymodel.Hash,
) int {
	amountOfQueryWords := 0
	for _, queryWord := range queryWords {
		if _, amongTheWords := words[queryWord]; !amongTheWords {
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
