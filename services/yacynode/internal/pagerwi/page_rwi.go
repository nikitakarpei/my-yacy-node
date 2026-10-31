// Package pagerwi turns a scraped page and the document it holds into the reverse word index
// the node stores: one posting per distinct word of the page's text that is no stopword of its
// language, and one URL metadata row that carries the postings root of those postings. A page
// without such a word has no reverse word index.
package pagerwi

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Stopwords interface {
	Contains(word yacymodel.Hash, language yacymodel.Language) bool
}

type IndexBuilder struct {
	stopwords  Stopwords
	emptyProof yacymodel.PostingsProof
}

func NewIndexBuilder(stopwords Stopwords, partitions yacymodel.DHTRingPartitions) IndexBuilder {
	return IndexBuilder{stopwords: stopwords, emptyProof: yacymodel.EmptyPostingsProof(partitions)}
}

type PageRWI struct {
	pagePostings yacymodel.PagePostings
	pageURL      canonicalurl.CanonicalURL
	metadata     yacymodel.URLMetadata
}

func (b IndexBuilder) RWIFrom(
	ctx context.Context,
	scrapedPage pagescrapecontract.OfferedPage,
	document documentextraction.Document,
	text []byte,
	reachedAt time.Time,
) yacymodel.Optional[PageRWI] {
	urlHash := yacymodel.URLHashOf(scrapedPage.PageURL.WebAddress())
	modifiedAt := recordedModifiedAtOf(scrapedPage, reachedAt)
	textWordsInOrder, textWordOccurrences, textStats := tokenize(string(text))
	_, titleWordOccurrences, _ := tokenize(document.Title)
	postings := b.postingsFrom(
		sharedPostingOf(scrapedPage.PageURL, document, modifiedAt, urlHash, textStats),
		textWordsInOrder,
		textWordOccurrences,
		titleWordOccurrences,
	)
	pagePostings, built := b.pagePostingsFrom(ctx, postings).Get()
	if !built {
		return yacymodel.None[PageRWI]()
	}

	return yacymodel.Some(PageRWI{
		pageURL: scrapedPage.PageURL,
		metadata: metadataOf(
			scrapedPage,
			document,
			reachedAt,
			modifiedAt,
			textStats.Words,
			pagePostings,
		),
		pagePostings: pagePostings,
	})
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

func sharedPostingOf(
	pageURL canonicalurl.CanonicalURL,
	document documentextraction.Document,
	modifiedAt time.Time,
	urlHash yacymodel.URLHash,
	textStats textStatistics,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		URLHash:       urlHash,
		LastModified:  yacymodel.MicroDateOf(modifiedAt).OrElse(0),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      recordedLanguageOf(document),
		TitleWords:    yacymodel.TitleWordsOf(document.Title),
		TextWords:     yacymodel.SaturatedCardinalFrom[uint16](textStats.Words),
		Phrases:       yacymodel.SaturatedCardinalFrom[uint16](textStats.Phrases),
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

func (b IndexBuilder) postingsFrom(
	shared yacymodel.RWIPosting,
	textWordsInOrder []string,
	textWordOccurrences map[string]wordOccurrence,
	titleWordOccurrences map[string]wordOccurrence,
) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(textWordsInOrder))
	postedWords := make(map[yacymodel.Hash]struct{}, len(textWordsInOrder))
	for _, word := range textWordsInOrder {
		wordHash := yacymodel.WordHash(word)
		if _, posted := postedWords[wordHash]; posted ||
			b.stopwords.Contains(wordHash, shared.Language) {
			continue
		}
		postedWords[wordHash] = struct{}{}
		posting := shared
		posting.WordHash = wordHash
		occurrenceInText := textWordOccurrences[word]
		_, appearsInTitle := titleWordOccurrences[word]
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

	return postings
}

func (b IndexBuilder) pagePostingsFrom(
	ctx context.Context,
	postings []yacymodel.RWIPosting,
) yacymodel.Optional[yacymodel.PagePostings] {
	if len(postings) == 0 {
		return yacymodel.None[yacymodel.PagePostings]()
	}
	// A crawl holds every posting of the page, so no hash is missing from its proof.
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(postings, b.emptyProof)
	if err != nil {
		slog.WarnContext(ctx, "page rwi not built",
			slog.String("url", postings[0].URLHash.String()),
			slog.Any("error", err),
		)

		return yacymodel.None[yacymodel.PagePostings]()
	}

	return yacymodel.Some(pagePostings)
}

//nolint:revive // argument-limit: each argument is a distinct source of one metadata row
func metadataOf(
	scrapedPage pagescrapecontract.OfferedPage,
	document documentextraction.Document,
	reachedAt time.Time,
	modifiedAt time.Time,
	wordCount int,
	pagePostings yacymodel.PagePostings,
) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Hash:         pagePostings.URL(),
		Address:      scrapedPage.PageURL.String(),
		Title:        document.Title,
		Modified:     yacymodel.Some(yacymodel.CalendarDayOf(modifiedAt)),
		Loaded:       yacymodel.Some(yacymodel.CalendarDayOf(reachedAt)),
		DocumentType: yacymodel.DocumentTypeText,
		Language:     declaredLanguageOf(document),
		PostingsRoot: pagePostings.Root(),
		ByteSize: yacymodel.Some(
			yacymodel.SaturatedCardinalFrom[uint32](len(scrapedPage.Body)),
		),
		WordCount:     yacymodel.SaturatedCardinalFrom[uint32](wordCount),
		LocalLinks:    yacymodel.SaturatedCardinalFrom[uint32](document.LocalLinks),
		ExternalLinks: yacymodel.SaturatedCardinalFrom[uint32](document.ExternalLinks),
	}
}

func (r PageRWI) PageURL() canonicalurl.CanonicalURL {
	return r.pageURL
}

func (r PageRWI) Metadata() yacymodel.URLMetadata {
	return r.metadata
}

func (r PageRWI) PagePostings() yacymodel.PagePostings {
	return r.pagePostings
}

func (r PageRWI) AmountOfPostings() int {
	return r.pagePostings.AmountOfPostings()
}
