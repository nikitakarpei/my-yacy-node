// Package pagerwi turns a scraped page and the document it holds into the reverse word index
// the node stores: one URL metadata row and one posting per distinct word of the page's text.
package pagerwi

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageRWI struct {
	PageURL  canonicalurl.CanonicalURL
	Metadata yacymodel.URLMetadata
	Postings []yacymodel.RWIPosting
}

func Of(
	scrapedPage pagescrapecontract.OfferedPage,
	document documentextraction.Document,
	text []byte,
	reachedAt time.Time,
) PageRWI {
	pageURL := scrapedPage.PageURL
	modifiedAt := recordedModifiedAtOf(scrapedPage, reachedAt)

	textWordsInOrder, textWordOccurrences, textStats := tokenize(string(text))
	_, titleWordOccurrences, _ := tokenize(document.Title)

	metadata := metadataOf(scrapedPage, document, reachedAt, modifiedAt, textStats.Words)
	shared := sharedPosting(pageURL, document, modifiedAt, metadata.Hash)
	shared.TitleWords = yacymodel.TitleWordsOf(document.Title)
	shared.TextWords = yacymodel.SaturatedCardinalFrom[uint16](textStats.Words)
	shared.Phrases = yacymodel.SaturatedCardinalFrom[uint16](textStats.Phrases)

	postings := make([]yacymodel.RWIPosting, 0, len(textWordsInOrder))
	for _, word := range textWordsInOrder {
		occurrenceInText := textWordOccurrences[word]
		_, appearsInTitle := titleWordOccurrences[word]
		posting := shared
		posting.WordHash = yacymodel.WordHash(word)
		posting.Appearance.AppearsInTitle = appearsInTitle
		posting.Hits = yacymodel.WrappedCardinalFrom(occurrenceInText.count)
		posting.TextPosition = yacymodel.SaturatedCardinalFrom[uint16](
			occurrenceInText.firstPosition,
		)
		posting.PhraseRelativePosition = yacymodel.WrappedCardinalFrom(
			occurrenceInText.firstPositionInPhrase,
		)
		posting.PhrasePosition = yacymodel.WrappedCardinalFrom(occurrenceInText.firstPhraseNumber)
		postings = append(postings, posting)
	}

	return PageRWI{
		PageURL:  pageURL,
		Metadata: metadata,
		Postings: postings,
	}
}

func recordedModifiedAtOf(
	scrapedPage pagescrapecontract.OfferedPage,
	reachedAt time.Time,
) time.Time {
	declared := scrapedPage.PageModifiedAt
	if declared.IsZero() || declared.After(reachedAt) ||
		!yacymodel.MicroDateOf(declared).Present() {
		return reachedAt
	}

	return declared
}

func metadataOf(
	scrapedPage pagescrapecontract.OfferedPage,
	document documentextraction.Document,
	reachedAt time.Time,
	modifiedAt time.Time,
	wordCount int,
) yacymodel.URLMetadata {
	pageURL := scrapedPage.PageURL

	return yacymodel.URLMetadata{
		Hash:         yacymodel.URLHashOf(pageURL.WebAddress()),
		Address:      pageURL.String(),
		Title:        document.Title,
		Modified:     yacymodel.Some(yacymodel.CalendarDayOf(modifiedAt)),
		Loaded:       yacymodel.Some(yacymodel.CalendarDayOf(reachedAt)),
		DocumentType: yacymodel.DocumentTypeText,
		Language:     declaredLanguageOf(document),
		ByteSize: yacymodel.Some(
			yacymodel.SaturatedCardinalFrom[uint32](len(scrapedPage.Body)),
		),
		WordCount:     yacymodel.SaturatedCardinalFrom[uint32](wordCount),
		LocalLinks:    yacymodel.SaturatedCardinalFrom[uint32](document.LocalLinks),
		ExternalLinks: yacymodel.SaturatedCardinalFrom[uint32](document.ExternalLinks),
	}
}

func declaredLanguageOf(
	document documentextraction.Document,
) yacymodel.Optional[yacymodel.Language] {
	if document.Language == "" {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(document.Language)
	if err != nil {
		return yacymodel.None[yacymodel.Language]()
	}

	return yacymodel.Some(language)
}

func sharedPosting(
	pageURL canonicalurl.CanonicalURL,
	document documentextraction.Document,
	modifiedAt time.Time,
	urlHash yacymodel.URLHash,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:       urlHash,
		LastModified:  yacymodel.MicroDateOf(modifiedAt).OrElse(0),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      recordedLanguageOf(document),
		LocalLinks:    yacymodel.WrappedCardinalFrom(document.LocalLinks),
		ExternalLinks: yacymodel.WrappedCardinalFrom(document.ExternalLinks),
		URLLength:     yacymodel.WrappedCardinalFrom(len(pageURL.String())),
		URLComponents: yacymodel.URLComponentsOf(pageURL.String()),
	}
}

func recordedLanguageOf(document documentextraction.Document) yacymodel.Language {
	declared, ok := declaredLanguageOf(document).Get()
	if !ok {
		return yacymodel.LanguageOfUndeclaredDocument
	}

	return declared
}
