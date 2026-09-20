package judgedqueries_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	recordedAnswersDirectory  = "testdata/answers"
	recordedAnswersFileSuffix = ".json.gz"
	fixtureFilePermissions    = 0o644
	fixtureDirPermissions     = 0o755
)

type recordedAnswers struct {
	Query                     string                  `json:"query"`
	RecordedAt                time.Time               `json:"recordedAt"`
	FoundDocuments            []recordedFoundDocument `json:"foundDocuments"`
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int  `json:"documentsHeldPerQueryWord"`
}

type recordedFoundDocument struct {
	Hash             yacymodel.URLHash       `json:"hash"`
	Address          string                  `json:"address"`
	Title            string                  `json:"title"`
	Snippet          string                  `json:"snippet"`
	HitsPerQueryWord map[yacymodel.Hash]int  `json:"hitsPerQueryWord"`
	AmountOfWords    int                     `json:"amountOfWords"`
	QueryPhraseHits  int                     `json:"queryPhraseHits"`
	AmountOfLinks    yacymodel.Optional[int] `json:"amountOfLinks,omitempty"`
}

func (r recordedFoundDocument) amountOfWordsOfTheReadPage() yacymodel.Optional[int] {
	if !r.holdsTheFactsOfAPageRead() || r.AmountOfWords <= 0 {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(r.AmountOfWords)
}

func (r recordedFoundDocument) queryPhraseHitsOfTheReadPage() yacymodel.Optional[int] {
	if !r.holdsTheFactsOfAPageRead() {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(r.QueryPhraseHits)
}

func (r recordedFoundDocument) holdsTheFactsOfAPageRead() bool {
	return r.AmountOfLinks.Present()
}

func (r recordedAnswers) answeredQuery() queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:                searchquery.QueryFrom(r.Query, "").TermHashes(),
		FoundDocuments:            foundDocumentsFrom(r.FoundDocuments),
		FactsPerDocument:          factsPerDocumentFrom(r.FoundDocuments),
		DocumentsHeldPerQueryWord: r.DocumentsHeldPerQueryWord,
	}
}

func foundDocumentsFrom(
	recordedFoundDocuments []recordedFoundDocument,
) []queryanswers.FoundDocument {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(recordedFoundDocuments))
	for _, recorded := range recordedFoundDocuments {
		foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
			Hash:    recorded.Hash,
			Address: recorded.Address,
			Title:   recorded.Title,
			Snippet: recorded.Snippet,
		})
	}

	return foundDocuments
}

func factsPerDocumentFrom(
	recordedFoundDocuments []recordedFoundDocument,
) queryanswers.FactsPerDocument {
	factsPerDocument := make(queryanswers.FactsPerDocument, len(recordedFoundDocuments))
	for _, recorded := range recordedFoundDocuments {
		factsPerDocument[recorded.Hash] = queryanswers.DocumentFacts{
			HitsPerQueryWord: recorded.HitsPerQueryWord,
			QueryPhraseHits:  recorded.queryPhraseHitsOfTheReadPage(),
			AmountOfWords:    recorded.amountOfWordsOfTheReadPage(),
			AmountOfLinks:    recorded.AmountOfLinks,
		}
	}

	return factsPerDocument
}

func recordedAnswersOf(query string, answers queryanswers.AnsweredQuery) recordedAnswers {
	return recordedAnswers{
		Query:                     query,
		RecordedAt:                time.Now().UTC().Truncate(time.Second),
		FoundDocuments:            recordedFoundDocumentsFrom(answers),
		DocumentsHeldPerQueryWord: answers.DocumentsHeldPerQueryWord,
	}
}

func recordedFoundDocumentsFrom(
	answers queryanswers.AnsweredQuery,
) []recordedFoundDocument {
	recordedFoundDocuments := make([]recordedFoundDocument, 0, len(answers.FoundDocuments))
	for _, foundDocument := range answers.FoundDocuments {
		facts := answers.FactsPerDocument[foundDocument.Hash]
		recordedFoundDocuments = append(recordedFoundDocuments, recordedFoundDocument{
			Hash:             foundDocument.Hash,
			Address:          foundDocument.Address,
			Title:            foundDocument.Title,
			Snippet:          foundDocument.Snippet,
			HitsPerQueryWord: facts.HitsPerQueryWord,
			AmountOfWords:    facts.AmountOfWords.OrElse(0),
			QueryPhraseHits:  facts.QueryPhraseHits.OrElse(0),
			AmountOfLinks:    facts.AmountOfLinks,
		})
	}

	return recordedFoundDocuments
}

func recordedAnswersFiles(t *testing.T) []string {
	t.Helper()

	answersFiles, err := filepath.Glob(
		filepath.Join(recordedAnswersDirectory, "*"+recordedAnswersFileSuffix),
	)
	if err != nil {
		t.Fatalf("read %s: %v", recordedAnswersDirectory, err)
	}
	if len(answersFiles) == 0 {
		t.Fatalf("no query is recorded in %s", recordedAnswersDirectory)
	}

	return answersFiles
}

func recordedAnswersInTheFile(t *testing.T, path string) recordedAnswers {
	t.Helper()

	var answers recordedAnswers
	if err := json.Unmarshal(contentOfTheGzippedFixtureFile(t, path), &answers); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return answers
}

func writeRecordedAnswersFile(t *testing.T, path string, answers recordedAnswers) {
	t.Helper()

	content, err := json.MarshalIndent(answers, "", "  ")
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	writeGzippedFixtureFile(t, path, append(content, '\n'))
}

func writeFixtureFile(t *testing.T, path string, fixture any) {
	t.Helper()

	content, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(content, '\n'), fixtureFilePermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
