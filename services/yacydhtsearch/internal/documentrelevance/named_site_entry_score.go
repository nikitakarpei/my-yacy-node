package documentrelevance

import (
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const prefixOfAWorldWideWebHost = "www."

func namedSiteEntryScoreOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) float64 {
	readAddress, err := url.Parse(foundDocument.Address)
	if err != nil {
		return 0
	}
	wordsOfTheSiteName := wordsOfTheSiteNameOf(readAddress.Hostname())

	amountOfQueryWordsInTheSiteName := 0
	for _, word := range queryWords {
		if _, inTheSiteName := wordsOfTheSiteName[word]; !inTheSiteName {
			continue
		}
		amountOfQueryWordsInTheSiteName++
	}
	if amountOfQueryWordsInTheSiteName == 0 {
		return 0
	}

	return float64(amountOfQueryWordsInTheSiteName) / float64(len(wordsOfTheSiteName)) *
		float64(amountOfQueryWordsInTheSiteName) / float64(len(queryWords)) /
		float64(1+amountOfPathSegmentsOf(readAddress.Path))
}

func wordsOfTheSiteNameOf(host string) map[yacymodel.Hash]struct{} {
	siteName := strings.TrimPrefix(host, prefixOfAWorldWideWebHost)
	if placeOfTheLastDot := strings.LastIndex(siteName, "."); placeOfTheLastDot >= 0 {
		siteName = siteName[:placeOfTheLastDot]
	}
	spelledWords := yacymodel.WordsIn(siteName)

	wordsOfTheSiteName := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		wordsOfTheSiteName[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return wordsOfTheSiteName
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
