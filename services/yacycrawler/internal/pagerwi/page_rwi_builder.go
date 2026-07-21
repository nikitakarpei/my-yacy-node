package pagerwi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/htmltext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func Build(
	page crawlcapability.CrawledPage,
	document []byte,
) (yacycrawlcontract.PageRWIRepresentation, error) {
	urlHash, err := yacymodel.HashURL(page.CanonicalURL)
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf("hash url: %w", err)
	}

	text, err := htmltext.Flatten(document)
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf("flatten document: %w", err)
	}

	order, occurrences, textStats := tokenize(text)
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
		Metadata: []yacymodel.URLMetadata{
			metadataOf(page, len(document), textStats.Words),
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
		Language:      languageOf(page),
		LocalLinks:    page.LocalLinkCount,
		ExternalLinks: page.ExternalLinkCount,
		URLLength:     len(page.CanonicalURL),
		URLComponents: componentCount(page.CanonicalURL),
	}
}

func metadataOf(
	page crawlcapability.CrawledPage,
	textLength int,
	wordCount int,
) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:       page.CanonicalURL,
		Title:         page.Title,
		Loaded:        yacymodel.Some(yacymodel.CalendarDayOf(page.CrawledAt)),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(page),
		ByteSize:      textLength,
		WordCount:     wordCount,
		LocalLinks:    page.LocalLinkCount,
		ExternalLinks: page.ExternalLinkCount,
	}
}

func languageOf(page crawlcapability.CrawledPage) yacymodel.Optional[yacymodel.Language] {
	if page.Language == "" {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(page.Language)
	if err != nil {
		return yacymodel.None[yacymodel.Language]()
	}

	return yacymodel.Some(language)
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
