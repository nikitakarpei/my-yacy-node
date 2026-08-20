package rwi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func buildRepresentation(
	page pagepublication.Page,
	fullText []byte,
) (yacycrawlcontract.PageRWIRepresentation, error) {
	canonicalAddress, err := url.Parse(page.CanonicalURL.String())
	if err != nil {
		return yacycrawlcontract.PageRWIRepresentation{}, fmt.Errorf("parse canonical url: %w", err)
	}
	urlHash := yacymodel.URLNormalformOf(canonicalAddress).Hash()

	order, occurrences, textStats := tokenize(string(fullText))
	_, _, titleStats := tokenize(page.Title)

	shared := sharedPosting(page, urlHash, canonicalAddress.Path)
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
			metadataOf(page, len(page.Body), textStats.Words),
		},
		Postings: postings,
	}, nil
}

func sharedPosting(
	page pagepublication.Page,
	urlHash yacymodel.URLHash,
	canonicalPath string,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:       urlHash,
		LastModified:  yacymodel.MicroDateFromTime(page.CrawledAt),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(page),
		LocalLinks:    page.LocalLinks,
		ExternalLinks: page.ExternalLinks,
		URLLength:     len(page.CanonicalURL.String()),
		URLComponents: componentCount(canonicalPath),
	}
}

func metadataOf(
	page pagepublication.Page,
	textLength int,
	wordCount int,
) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:       page.CanonicalURL.String(),
		Title:         page.Title,
		Loaded:        yacymodel.Some(yacymodel.CalendarDayOf(page.CrawledAt)),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(page),
		ByteSize:      textLength,
		WordCount:     wordCount,
		LocalLinks:    page.LocalLinks,
		ExternalLinks: page.ExternalLinks,
	}
}

func languageOf(page pagepublication.Page) yacymodel.Optional[yacymodel.Language] {
	if page.Language == "" {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(page.Language)
	if err != nil {
		return yacymodel.None[yacymodel.Language]()
	}

	return yacymodel.Some(language)
}

func componentCount(canonicalPath string) int {
	trimmed := strings.Trim(canonicalPath, "/")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "/") + 1
}
