package documentrelevance

import (
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	prefixOfWorldWideWebHost = "www."

	namedSiteEntryScoreOfMalformedAddress = 0.0
	namedSiteEntryScoreOfOtherSite        = 0.0

	amountOfStepsToSiteEntry = 1
)

type namedSiteEntryScorer struct {
	queryWords []yacymodel.Hash
}

func namedSiteEntryScorerFrom(statistics answersStatistics) namedSiteEntryScorer {
	return namedSiteEntryScorer{queryWords: statistics.queryWords}
}

func (scorer namedSiteEntryScorer) scoreOf(document queryanswers.FoundDocument) float64 {
	address, err := url.Parse(document.Address)
	if err != nil {
		return namedSiteEntryScoreOfMalformedAddress
	}
	wordsInSiteName := wordsInSiteNameOf(address.Hostname())
	amountOfQueryWordsInSiteName := scorer.amountOfQueryWordsAmong(wordsInSiteName)
	if amountOfQueryWordsInSiteName == 0 {
		return namedSiteEntryScoreOfOtherSite
	}

	shareOfSiteNameTheQueryHolds := float64(amountOfQueryWordsInSiteName) /
		float64(len(wordsInSiteName))
	shareOfQueryTheSiteNameHolds := float64(amountOfQueryWordsInSiteName) /
		float64(len(scorer.queryWords))
	amountOfStepsToDocument := amountOfStepsToSiteEntry +
		amountOfPathSegmentsOf(address.Path)

	return shareOfSiteNameTheQueryHolds * shareOfQueryTheSiteNameHolds /
		float64(amountOfStepsToDocument)
}

func wordsInSiteNameOf(host string) map[yacymodel.Hash]struct{} {
	siteName := strings.TrimPrefix(host, prefixOfWorldWideWebHost)
	if placeOfLastDot := strings.LastIndex(siteName, "."); placeOfLastDot >= 0 {
		siteName = siteName[:placeOfLastDot]
	}

	return wordsIn(siteName)
}

func (scorer namedSiteEntryScorer) amountOfQueryWordsAmong(
	words map[yacymodel.Hash]struct{},
) int {
	amountOfQueryWords := 0
	for _, queryWord := range scorer.queryWords {
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
