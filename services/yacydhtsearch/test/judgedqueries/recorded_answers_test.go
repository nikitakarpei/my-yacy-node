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
	Hash            yacymodel.URLHash                    `json:"hash"`
	Address         string                               `json:"address"`
	Title           string                               `json:"title"`
	Snippet         string                               `json:"snippet"`
	MatchedWords    map[yacymodel.Hash]recordedWordCount `json:"matchedWords"`
	QueryPhraseHits int                                  `json:"queryPhraseHits"`
}

type recordedWordCount struct {
	Hits      int `json:"hits"`
	TextWords int `json:"textWords"`
}

func (r recordedAnswers) answeredQuery() queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:                searchquery.QueryFrom(r.Query, "").TermHashes(),
		FoundDocuments:            foundDocumentsFrom(r.FoundDocuments),
		DocumentsHeldPerQueryWord: r.DocumentsHeldPerQueryWord,
	}
}

func foundDocumentsFrom(
	recordedFoundDocuments []recordedFoundDocument,
) []queryanswers.FoundDocument {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(recordedFoundDocuments))
	for _, recorded := range recordedFoundDocuments {
		foundDocuments = append(foundDocuments, queryanswers.FoundDocument{
			Metadata: yacymodel.URLMetadata{
				Hash:    recorded.Hash,
				Address: recorded.Address,
				Title:   recorded.Title,
				Snippet: recorded.Snippet,
			},
			MatchedWords:    wordCountsOf(recorded.MatchedWords),
			QueryPhraseHits: recorded.QueryPhraseHits,
		})
	}

	return foundDocuments
}

func wordCountsOf(
	recordedWordCounts map[yacymodel.Hash]recordedWordCount,
) map[yacymodel.Hash]queryanswers.WordCount {
	wordCounts := make(map[yacymodel.Hash]queryanswers.WordCount, len(recordedWordCounts))
	for word, recorded := range recordedWordCounts {
		wordCounts[word] = queryanswers.WordCount{
			Hits:      recorded.Hits,
			TextWords: recorded.TextWords,
		}
	}

	return wordCounts
}

func recordedAnswersOf(query string, answers queryanswers.AnsweredQuery) recordedAnswers {
	return recordedAnswers{
		Query:                     query,
		RecordedAt:                time.Now().UTC().Truncate(time.Second),
		FoundDocuments:            recordedFoundDocumentsFrom(answers.FoundDocuments),
		DocumentsHeldPerQueryWord: answers.DocumentsHeldPerQueryWord,
	}
}

func recordedFoundDocumentsFrom(
	foundDocuments []queryanswers.FoundDocument,
) []recordedFoundDocument {
	recordedFoundDocuments := make([]recordedFoundDocument, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		recordedFoundDocuments = append(recordedFoundDocuments, recordedFoundDocument{
			Hash:            foundDocument.Metadata.Hash,
			Address:         foundDocument.Metadata.Address,
			Title:           foundDocument.Metadata.Title,
			Snippet:         foundDocument.Metadata.Snippet,
			MatchedWords:    recordedWordCountsOf(foundDocument.MatchedWords),
			QueryPhraseHits: foundDocument.QueryPhraseHits,
		})
	}

	return recordedFoundDocuments
}

func recordedWordCountsOf(
	wordCounts map[yacymodel.Hash]queryanswers.WordCount,
) map[yacymodel.Hash]recordedWordCount {
	recordedWordCounts := make(map[yacymodel.Hash]recordedWordCount, len(wordCounts))
	for word, wordCount := range wordCounts {
		recordedWordCounts[word] = recordedWordCount{
			Hits:      wordCount.Hits,
			TextWords: wordCount.TextWords,
		}
	}

	return recordedWordCounts
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
	if err := json.Unmarshal(contentOfTheCompressedFixtureFile(t, path), &answers); err != nil {
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
	writeCompressedFixtureFile(t, path, append(content, '\n'))
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
