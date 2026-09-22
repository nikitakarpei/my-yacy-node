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
	queryVocabulary queryVocabulary
}

func namedSiteEntryScorerFrom(statistics answersStatistics) namedSiteEntryScorer {
	return namedSiteEntryScorer{queryVocabulary: statistics.queryVocabulary}
}

func (scorer namedSiteEntryScorer) scoreOf(document queryanswers.FoundDocument) float64 {
	address, err := url.Parse(document.Address)
	if err != nil {
		return namedSiteEntryScoreOfMalformedAddress
	}
	siteName := siteNameOf(address.Hostname())
	amountOfWordsInSiteName := len(yacymodel.WordsIn(siteName))
	amountOfQueryWordsInSiteName := len(scorer.queryVocabulary.wordsIn(siteName))
	if amountOfQueryWordsInSiteName == 0 {
		return namedSiteEntryScoreOfOtherSite
	}

	shareOfSiteNameTheQueryHolds := float64(amountOfQueryWordsInSiteName) /
		float64(amountOfWordsInSiteName)
	shareOfQueryTheSiteNameHolds := float64(amountOfQueryWordsInSiteName) /
		float64(len(scorer.queryVocabulary.words))
	amountOfStepsToDocument := amountOfStepsToSiteEntry +
		amountOfPathSegmentsOf(address.Path)

	return shareOfSiteNameTheQueryHolds * shareOfQueryTheSiteNameHolds /
		float64(amountOfStepsToDocument)
}

func siteNameOf(host string) string {
	siteName := strings.TrimPrefix(host, prefixOfWorldWideWebHost)
	if placeOfLastDot := strings.LastIndex(siteName, "."); placeOfLastDot >= 0 {
		siteName = siteName[:placeOfLastDot]
	}

	return siteName
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
