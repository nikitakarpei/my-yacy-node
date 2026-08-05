package postingfilter

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
)

func urlHashFor(url string) yacymodel.URLHash {
	const filler = "AAAAAAAAAAAA"
	padded := url + filler[len(url):]
	wordHash, err := yacymodel.ParseHash(padded)
	if err != nil {
		panic(err)
	}
	hash, err := yacymodel.ParseURLHash(wordHash.String())
	if err != nil {
		panic(err)
	}

	return hash
}

func postingOfType(kind yacymodel.DocumentType) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{DocumentType: kind}
}

func postingWith(appearance yacymodel.Appearance) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{Appearance: appearance}
}

func TestIsOfDocumentType(t *testing.T) {
	if !isOfDocumentType(
		postingOfType(yacymodel.DocumentTypeImage),
		searchcriteria.ImageContent,
	) {
		t.Fatal("image document should match strict image")
	}
	if isOfDocumentType(
		postingOfType(yacymodel.DocumentTypeAudio),
		searchcriteria.ImageContent,
	) {
		t.Fatal("audio document should not match strict image")
	}
	if !isOfDocumentType(
		postingOfType(yacymodel.DocumentTypeMovie),
		searchcriteria.VideoContent,
	) {
		t.Fatal("movie document should match strict video")
	}
	if !isOfDocumentType(
		postingWith(yacymodel.Appearance{HasApp: true}),
		searchcriteria.ApplicationContent,
	) {
		t.Fatal("app appearance should match app")
	}
	if !isOfDocumentType(
		postingOfType(yacymodel.DocumentTypeImage),
		searchcriteria.AnyContent,
	) {
		t.Fatal("any content kind should pass through")
	}
}

func TestAppearsAsContentKind(t *testing.T) {
	if !appearsAsContentKind(
		postingWith(yacymodel.Appearance{HasAudio: true}),
		searchcriteria.AudioContent,
	) {
		t.Fatal("audio appearance should match loose audio")
	}
	if appearsAsContentKind(
		postingWith(yacymodel.Appearance{HasImage: true}),
		searchcriteria.AudioContent,
	) {
		t.Fatal("image appearance should not match loose audio")
	}
	if !appearsAsContentKind(
		postingWith(yacymodel.Appearance{HasVideo: true}),
		searchcriteria.VideoContent,
	) {
		t.Fatal("video appearance should match loose video")
	}
	if !appearsAsContentKind(
		postingWith(yacymodel.Appearance{HasImage: true}),
		searchcriteria.ImageContent,
	) {
		t.Fatal("image appearance should match loose image")
	}
	if !appearsAsContentKind(
		postingWith(yacymodel.Appearance{HasApp: true}),
		searchcriteria.ApplicationContent,
	) {
		t.Fatal("app appearance should match app")
	}
	if !appearsAsContentKind(
		postingOfType(yacymodel.DocumentTypeImage),
		searchcriteria.AnyContent,
	) {
		t.Fatal("any content kind should pass through")
	}
}

func TestIsFromRequestedSite(t *testing.T) {
	documentHash, err := yacymodel.ParseURLHash("0123456789AB")
	if err != nil {
		t.Fatalf("parse url hash: %v", err)
	}
	if !isFromRequestedSite(documentHash, yacymodel.None[yacymodel.HostHash]()) {
		t.Fatal("empty site hash should match")
	}
	if !isFromRequestedSite(documentHash, yacymodel.Some(mustHostHash(t, "6789AB"))) {
		t.Fatal("matching host hash should match")
	}
	if isFromRequestedSite(documentHash, yacymodel.Some(mustHostHash(t, "000000"))) {
		t.Fatal("non-matching host hash should not match")
	}
}

func mustHostHash(t *testing.T, s string) yacymodel.HostHash {
	t.Helper()
	hash, err := yacymodel.ParseHostHash(s)
	if err != nil {
		t.Fatalf("ParseHostHash(%q): %v", s, err)
	}

	return hash
}

func TestSharesRequiredAppearance(t *testing.T) {
	posting := postingWith(yacymodel.Appearance{HasImage: true})

	if !sharesRequiredAppearance(posting, yacymodel.None[yacymodel.Appearance]()) {
		t.Fatal("no required properties should match")
	}
	if !sharesRequiredAppearance(
		posting,
		yacymodel.Some(yacymodel.Appearance{HasImage: true}),
	) {
		t.Fatal("required property present in appearance should match")
	}
	if sharesRequiredAppearance(
		posting,
		yacymodel.Some(yacymodel.Appearance{HasVideo: true}),
	) {
		t.Fatal("required property absent from appearance should not match")
	}
}

func TestDocumentSetHoldsEveryDocument(t *testing.T) {
	if documentSet(nil) != nil {
		t.Fatal("nil input should return nil")
	}
	first, second := urlHashFor("url-a"), urlHashFor("url-b")
	set := documentSet([]yacymodel.URLHash{first, second})
	if _, ok := set[first]; !ok {
		t.Fatal("first identifier missing")
	}
	if _, ok := set[second]; !ok {
		t.Fatal("second identifier missing")
	}
}

type fakeScanner struct {
	postings map[yacymodel.Hash][]yacymodel.RWIPosting
}

func (s fakeScanner) RWICount(context.Context) (int, error) {
	return len(s.postings), nil
}

func (s fakeScanner) ScanWord(
	_ context.Context,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	for _, entry := range s.postings[word] {
		entry.WordHash = word
		keepGoing, err := visit(entry)
		if err != nil {
			return err
		}
		if !keepGoing {
			return nil
		}
	}

	return nil
}

func (s fakeScanner) PostingOf(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	for _, entry := range s.postings[word] {
		if entry.URLHash == url {
			entry.WordHash = word

			return entry, true, nil
		}
	}

	return yacymodel.RWIPosting{}, false, nil
}

func wordHashFor(base string) yacymodel.Hash {
	const filler = "AAAAAAAAAAAA"
	hash, err := yacymodel.ParseHash(base + filler[len(base):])
	if err != nil {
		panic(err)
	}

	return hash
}

func TestFilterForSearchRejectsDocumentsHoldingAnExcludedTerm(t *testing.T) {
	banned := wordHashFor("ban")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		banned: {{URLHash: urlHashFor("url-b")}},
	}}

	filter, err := FilterForSearch(
		context.Background(),
		index,
		searchcriteria.Criteria{ExcludedTerms: []yacymodel.Hash{banned}},
	)
	if err != nil {
		t.Fatalf("FilterForSearch: %v", err)
	}
	if !filter.Accepts(yacymodel.RWIPosting{URLHash: urlHashFor("url-a")}) {
		t.Error("document without the excluded term should be accepted")
	}
	if filter.Accepts(yacymodel.RWIPosting{URLHash: urlHashFor("url-b")}) {
		t.Error("document holding the excluded term should be rejected")
	}
}

func TestFilterForReportRejectsUnrequiredDocuments(t *testing.T) {
	filter := FilterForReport(searchcriteria.Criteria{
		RequiredDocuments: []yacymodel.URLHash{urlHashFor("url-a")},
	})

	if !filter.Accepts(yacymodel.RWIPosting{URLHash: urlHashFor("url-a")}) {
		t.Error("required document should be accepted")
	}
	if filter.Accepts(yacymodel.RWIPosting{URLHash: urlHashFor("url-b")}) {
		t.Error("document outside the required set should be rejected")
	}
}

func TestFilterForReportRejectsOtherLanguages(t *testing.T) {
	english, err := yacymodel.ParseLanguage("en")
	if err != nil {
		t.Fatalf("ParseLanguage: %v", err)
	}
	german, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatalf("ParseLanguage: %v", err)
	}
	filter := FilterForReport(searchcriteria.Criteria{Language: yacymodel.Some(english)})

	if !filter.Accepts(yacymodel.RWIPosting{Language: yacymodel.Some(english)}) {
		t.Error("posting in the required language should be accepted")
	}
	if filter.Accepts(yacymodel.RWIPosting{Language: yacymodel.Some(german)}) {
		t.Error("posting in another language should be rejected")
	}
	if filter.Accepts(yacymodel.RWIPosting{}) {
		t.Error("posting without a language should be rejected")
	}
}

func TestFilterForReportRejectsOtherDocumentTypes(t *testing.T) {
	filter := FilterForReport(searchcriteria.Criteria{
		ContentKind:       searchcriteria.ImageContent,
		StrictContentKind: true,
	})

	if !filter.Accepts(postingOfType(yacymodel.DocumentTypeImage)) {
		t.Error("image document should be accepted under strict image")
	}
	if filter.Accepts(postingOfType(yacymodel.DocumentTypeAudio)) {
		t.Error("audio document should be rejected under strict image")
	}
}
