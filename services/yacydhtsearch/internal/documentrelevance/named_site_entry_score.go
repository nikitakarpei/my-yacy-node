package documentrelevance

import (
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	prefixOfWorldWideWebHost = "www."

	namedSiteEntryScoreOfMalformedAddress = 0.0
	namedSiteEntryScoreOfOtherSite        = 0.0

	scoreShareOfSiteNamedByItsRegistrableDomain = 1.0
	scoreShareOfSiteNamedByASubdomainAlone      = 0.2

	amountOfStepsToSiteEntry = 1
)

type namedSiteEntryScorer struct {
	queryVocabulary queryVocabulary
}

func namedSiteEntryScorerFrom(statistics findingsStatistics) namedSiteEntryScorer {
	return namedSiteEntryScorer{queryVocabulary: statistics.queryVocabulary}
}

func (scorer namedSiteEntryScorer) scoreOf(document queryfindings.FoundDocument) float64 {
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

	return scorer.scoreShareOfSiteAt(address.Hostname()) *
		shareOfSiteNameTheQueryHolds * shareOfQueryTheSiteNameHolds /
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

func (scorer namedSiteEntryScorer) scoreShareOfSiteAt(host string) float64 {
	if len(scorer.queryVocabulary.wordsIn(registrableNameOf(host))) == 0 {
		return scoreShareOfSiteNamedByASubdomainAlone
	}

	return scoreShareOfSiteNamedByItsRegistrableDomain
}

func registrableNameOf(host string) string {
	underPublicSuffix, _ := strings.CutSuffix(host, "."+icannPublicSuffixOf(host))

	return underPublicSuffix[strings.LastIndex(underPublicSuffix, ".")+1:]
}

func icannPublicSuffixOf(host string) string {
	publicSuffix, icann := publicsuffix.PublicSuffix(host)
	for !icann && strings.Contains(publicSuffix, ".") {
		_, parentSuffix, _ := strings.Cut(publicSuffix, ".")
		publicSuffix, icann = publicsuffix.PublicSuffix(parentSuffix)
	}

	return publicSuffix
}
