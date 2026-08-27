// Package pagerwi turns a scraped page and the document it holds into the reverse word index
// the node stores: one URL metadata row and one posting per distinct word of the page's text.
package pagerwi

import (
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/scrapedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWI struct {
	PageURL  canonicalurl.CanonicalURL
	Metadata yacymodel.URLMetadata
	Postings []yacymodel.RWIPosting
}

func Of(
	scrapedPage scrapedpage.ScrapedPage,
	document documentextraction.Document,
	text []byte,
	reachedAt time.Time,
) PageRWI {
	pageURL := scrapedPage.PageURL
	urlHash := yacymodel.URLNormalformOf(pageURL.WebAddress()).Hash()

	order, occurrences, textStats := tokenize(string(text))
	_, _, titleStats := tokenize(document.Title)

	shared := sharedPosting(pageURL, document, reachedAt, urlHash)
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
		PageURL: pageURL,
		Metadata: metadataOf(
			pageURL, document, len(scrapedPage.Body), reachedAt, textStats.Words,
		),
		Postings: postings,
	}
}

func sharedPosting(
	pageURL canonicalurl.CanonicalURL,
	document documentextraction.Document,
	reachedAt time.Time,
	urlHash yacymodel.URLHash,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:       urlHash,
		LastModified:  yacymodel.MicroDateFromTime(reachedAt),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(document),
		LocalLinks:    document.LocalLinks,
		ExternalLinks: document.ExternalLinks,
		URLLength:     len(pageURL.String()),
		URLComponents: componentCount(pageURL.WebAddress().Path),
	}
}

func metadataOf(
	pageURL canonicalurl.CanonicalURL,
	document documentextraction.Document,
	documentByteSize int,
	reachedAt time.Time,
	wordCount int,
) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:       pageURL.String(),
		Title:         document.Title,
		Loaded:        yacymodel.Some(yacymodel.CalendarDayOf(reachedAt)),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      languageOf(document),
		ByteSize:      documentByteSize,
		WordCount:     wordCount,
		LocalLinks:    document.LocalLinks,
		ExternalLinks: document.ExternalLinks,
	}
}

func languageOf(document documentextraction.Document) yacymodel.Optional[yacymodel.Language] {
	if document.Language == "" {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(document.Language)
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
