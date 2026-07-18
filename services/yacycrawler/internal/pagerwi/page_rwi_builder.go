package pagerwi

import (
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	documentTypeText = 't'
	secondsPerDay    = 86400
)

func Build(
	page crawlcapability.CrawledPage,
	text []byte,
) (yacycrawlcontract.PageRWIRepresentation, error) {
	urlHash, err := yacymodel.HashURL(page.CanonicalURL)
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf("hash url: %w", err)
	}

	order, occurrences, textStats := tokenize(string(text))
	_, _, titleStats := tokenize(page.Title)
	dayNumber := dayNumberOf(page.CrawledAt)

	stats := documentWordStatistics{
		TextWordCount:  textStats.Words,
		TitleWordCount: titleStats.Words,
		PhraseCount:    textStats.Phrases,
	}
	shared := sharedProperties(page, urlHash.String(), stats, dayNumber)

	postings := make([]yacymodel.RWIPostingWireForm, 0, len(order))
	for _, word := range order {
		occurrence := occurrences[word]
		properties := map[string]string{}
		maps.Copy(properties, shared)
		properties[yacymodel.ColHitCount] = strconv.Itoa(occurrence.count)
		properties[yacymodel.ColTextPosition] = strconv.Itoa(occurrence.firstPosition)
		properties[yacymodel.ColPhraseRelativePos] = strconv.Itoa(occurrence.firstPositionInPhrase)
		properties[yacymodel.ColPhrasePosition] = strconv.Itoa(occurrence.firstPhraseNumber)
		postings = append(postings, yacymodel.RWIPostingWireForm{
			WordHash:   yacymodel.WordHash(word),
			Properties: properties,
		})
	}

	return yacycrawlcontract.PageRWIRepresentation{
		CanonicalURL: page.CanonicalURL,
		Metadata: []yacymodel.URIMetadataRow{
			metadataRowOf(page, urlHash, len(text), textStats.Words),
		},
		Postings: postings,
	}, nil
}

func metadataRowOf(
	page crawlcapability.CrawledPage,
	urlHash yacymodel.URLHash,
	textLength int,
	wordCount int,
) yacymodel.URIMetadataRow {
	return yacymodel.URIMetadataRow{Properties: map[string]string{
		yacymodel.URLMetaHash:           urlHash.String(),
		"dt":                            string(rune(documentTypeText)),
		"url":                           yacymodel.EncodeBase64WireForm(page.CanonicalURL),
		yacymodel.URLMetaColDescription: yacymodel.EncodeBase64WireForm(page.Title),
		"size":                          strconv.Itoa(textLength),
		"wc":                            strconv.Itoa(wordCount),
		"llocal":                        strconv.Itoa(page.LocalLinkCount),
		"lother":                        strconv.Itoa(page.ExternalLinkCount),
	}}
}

// documentWordStatistics holds the shared RWI counters derived from tokenizing a page,
// grouped because they always travel together into sharedProperties.
type documentWordStatistics struct {
	TextWordCount  int
	TitleWordCount int
	PhraseCount    int
}

func sharedProperties(
	page crawlcapability.CrawledPage,
	urlHash string,
	stats documentWordStatistics,
	dayNumber uint64,
) map[string]string {
	properties := map[string]string{
		yacymodel.ColURLHash:           urlHash,
		yacymodel.ColTextWordCount:     strconv.Itoa(stats.TextWordCount),
		yacymodel.ColTitleWordCount:    strconv.Itoa(stats.TitleWordCount),
		yacymodel.ColPhraseCount:       strconv.Itoa(stats.PhraseCount),
		yacymodel.ColDocType:           strconv.Itoa(documentTypeText),
		yacymodel.ColLocalLinkCount:    strconv.Itoa(page.LocalLinkCount),
		yacymodel.ColExternalLinkCount: strconv.Itoa(page.ExternalLinkCount),
		yacymodel.ColURLLength:         strconv.Itoa(len(page.CanonicalURL)),
		yacymodel.ColURLComponentCount: strconv.Itoa(componentCount(page.CanonicalURL)),
		yacymodel.ColLastModified:      strconv.FormatUint(dayNumber, 10),
		yacymodel.ColFreshUntil:        strconv.FormatUint(dayNumber, 10),
	}
	if page.Language != "" {
		properties[yacymodel.ColLanguage] = page.Language
	}
	return properties
}

func dayNumberOf(crawledAt time.Time) uint64 {
	seconds := crawledAt.Unix()
	if seconds < 0 {
		return 0
	}
	return uint64(seconds) / secondsPerDay
}

func componentCount(canonicalURL string) int {
	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return 0
	}
	trimmed := strings.Trim(parsed.Path, "/")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "/") + 1
}
