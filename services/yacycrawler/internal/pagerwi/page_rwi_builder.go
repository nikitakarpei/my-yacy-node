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
	page crawlcapability.ExtractedPage,
	text crawlcapability.ContentDerivation,
) (yacycrawlcontract.PageRWIRepresentation, error) {
	urlHash, err := yacymodel.HashURL(page.CanonicalURL)
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf("hash url: %w", err)
	}

	body, err := text.Derive(page.Body, page.Format)
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf("derive page text: %w", err)
	}

	order, occurrences, textStats := tokenize(string(body))
	_, _, titleStats := tokenize(page.Title)
	dayNumber := dayNumberOf(page.FetchedAt)

	stats := documentWordStatistics{
		TextWordCount:  textStats.Words,
		TitleWordCount: titleStats.Words,
		PhraseCount:    textStats.Phrases,
	}
	shared := sharedProperties(page, urlHash.String(), stats, dayNumber)

	postings := make([]yacymodel.RWIPosting, 0, len(order))
	for _, word := range order {
		occurrence := occurrences[word]
		properties := map[string]string{}
		maps.Copy(properties, shared)
		properties[yacymodel.ColHitCount] = strconv.Itoa(occurrence.count)
		properties[yacymodel.ColTextPosition] = strconv.Itoa(occurrence.firstPosition)
		properties[yacymodel.ColPhraseRelativePos] = strconv.Itoa(occurrence.firstPositionInPhrase)
		properties[yacymodel.ColPhrasePosition] = strconv.Itoa(occurrence.firstPhraseNumber)
		postings = append(postings, yacymodel.RWIPosting{
			WordHash:   yacymodel.WordHash(word),
			Properties: properties,
		})
	}

	metadata := metadataRow(page, urlHash.String(), len(body), textStats.Words)

	return yacycrawlcontract.PageRWIRepresentation{
		CanonicalURL: page.CanonicalURL,
		Metadata:     []yacymodel.URIMetadataRow{metadata},
		Postings:     postings,
	}, nil
}

// documentWordStatistics holds the shared RWI counters derived from tokenizing a page,
// grouped because they always travel together into sharedProperties.
type documentWordStatistics struct {
	TextWordCount  int
	TitleWordCount int
	PhraseCount    int
}

func sharedProperties(
	page crawlcapability.ExtractedPage,
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

func metadataRow(
	page crawlcapability.ExtractedPage,
	urlHash string,
	textLength int,
	total int,
) yacymodel.URIMetadataRow {
	return yacymodel.URIMetadataRow{Properties: map[string]string{
		yacymodel.URLMetaHash:           urlHash,
		"dt":                            string(rune(documentTypeText)),
		"url":                           yacymodel.EncodeBase64WireForm(page.CanonicalURL),
		yacymodel.URLMetaColDescription: yacymodel.EncodeBase64WireForm(page.Title),
		"size":                          strconv.Itoa(textLength),
		"wc":                            strconv.Itoa(total),
		"llocal":                        strconv.Itoa(page.LocalLinkCount),
		"lother":                        strconv.Itoa(page.ExternalLinkCount),
	}}
}

func dayNumberOf(fetchedAt time.Time) uint64 {
	seconds := fetchedAt.Unix()
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
