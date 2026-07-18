package pagerwi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const documentTypeText = 't'

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

	shared := sharedPosting(page, urlHash)
	shared.TitleWords = titleStats.Words
	shared.TextWords = textStats.Words
	shared.Phrases = textStats.Phrases

	postings := make([]yacymodel.RWIPosting, 0, len(order))
	for _, word := range order {
		occurrence := occurrences[word]
		posting := shared
		posting.WordHash = yacymodel.WordHash(word)
		posting.Hits = occurrence.count
		posting.TextPosition = occurrence.firstPosition
		posting.PhraseRelativePosition = occurrence.firstPositionInPhrase
		posting.PhrasePosition = occurrence.firstPhraseNumber
		postings = append(postings, posting)
	}

	return yacycrawlcontract.PageRWIRepresentation{
		CanonicalURL: page.CanonicalURL,
		Metadata: []yacymodel.URIMetadataRow{
			metadataRowOf(page, urlHash, len(text), textStats.Words),
		},
		Postings: postings,
	}, nil
}

func sharedPosting(
	page crawlcapability.CrawledPage,
	urlHash yacymodel.URLHash,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:       urlHash,
		LastModified:  yacymodel.MicroDateFromTime(page.CrawledAt),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      yacymodel.Language(page.Language),
		LocalLinks:    page.LocalLinkCount,
		ExternalLinks: page.ExternalLinkCount,
		URLLength:     len(page.CanonicalURL),
		URLComponents: componentCount(page.CanonicalURL),
	}
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
