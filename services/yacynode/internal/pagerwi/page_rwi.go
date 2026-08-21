// Package pagerwi turns a scraped page into the reverse word index the node stores:
// one URL metadata row and one posting per distinct word of the page's text.
package pagerwi

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWI struct {
	CanonicalURL string
	Metadata     yacymodel.URLMetadata
	Postings     []yacymodel.RWIPosting
}

func Of(page pagescrape.ScrapedPage, reachedAt time.Time) (PageRWI, error) {
	canonicalAddress, err := url.Parse(page.CanonicalURL)
	if err != nil {
		return PageRWI{}, fmt.Errorf("parse canonical url: %w", err)
	}
	urlHash := yacymodel.URLNormalformOf(canonicalAddress).Hash()

	order, occurrences, textStats := tokenize(string(page.Content))
	_, _, titleStats := tokenize(page.Title)

	shared := sharedPosting(page, reachedAt, urlHash)
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

	return PageRWI{
		CanonicalURL: page.CanonicalURL,
		Metadata:     metadataOf(page, reachedAt, textStats.Words),
		Postings:     postings,
	}, nil
}

func sharedPosting(
	page pagescrape.ScrapedPage,
	reachedAt time.Time,
	urlHash yacymodel.URLHash,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:       urlHash,
		LastModified:  yacymodel.MicroDateFromTime(reachedAt),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(page),
		LocalLinks:    page.LocalLinks,
		ExternalLinks: page.ExternalLinks,
		URLLength:     len(page.CanonicalURL),
		URLComponents: componentCount(page.CanonicalURL),
	}
}

func metadataOf(
	page pagescrape.ScrapedPage,
	reachedAt time.Time,
	wordCount int,
) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:       page.CanonicalURL,
		Title:         page.Title,
		Loaded:        yacymodel.Some(yacymodel.CalendarDayOf(reachedAt)),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(page),
		ByteSize:      page.DocumentByteSize,
		WordCount:     wordCount,
		LocalLinks:    page.LocalLinks,
		ExternalLinks: page.ExternalLinks,
	}
}

func languageOf(page pagescrape.ScrapedPage) yacymodel.Optional[yacymodel.Language] {
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
