package judgedqueries_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	recordedAnswersDirectory = "testdata/answers"
	fixtureFilePermissions   = 0o644
	fixtureDirPermissions    = 0o755
)

type recordedAnswers struct {
	Query                       string                 `json:"query"`
	RecordedAt                  time.Time              `json:"recordedAt"`
	ItemsInTheOrderOfEachAnswer [][]recordedItem       `json:"itemsInTheOrderOfEachAnswer"`
	ItemsInNoOrder              []recordedItem         `json:"itemsInNoOrder"`
	DocumentsHeldPerQueryWord   map[yacymodel.Hash]int `json:"documentsHeldPerQueryWord"`
}

type recordedItem struct {
	Hash         yacymodel.URLHash                    `json:"hash"`
	Address      string                               `json:"address"`
	Title        string                               `json:"title"`
	Snippet      string                               `json:"snippet"`
	MatchedWords map[yacymodel.Hash]recordedWordCount `json:"matchedWords"`
}

type recordedWordCount struct {
	Hits      int `json:"hits"`
	TextWords int `json:"textWords"`
}

func (r recordedAnswers) answeredQuery() peeranswers.AnsweredQuery {
	itemsInTheOrderOfEachAnswer := make(
		[][]peeranswers.AnsweredItem, 0, len(r.ItemsInTheOrderOfEachAnswer),
	)
	for _, recordedItemsOfOneAnswer := range r.ItemsInTheOrderOfEachAnswer {
		itemsInTheOrderOfEachAnswer = append(
			itemsInTheOrderOfEachAnswer, answeredItemsOf(recordedItemsOfOneAnswer),
		)
	}

	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachAnswer: itemsInTheOrderOfEachAnswer,
		ItemsInNoOrder:              answeredItemsOf(r.ItemsInNoOrder),
		DocumentsHeldPerQueryWord:   r.DocumentsHeldPerQueryWord,
	}
}

func answeredItemsOf(recordedItems []recordedItem) []peeranswers.AnsweredItem {
	answeredItems := make([]peeranswers.AnsweredItem, 0, len(recordedItems))
	for _, recorded := range recordedItems {
		answeredItems = append(answeredItems, peeranswers.AnsweredItem{
			Metadata: yacymodel.URLMetadata{
				Hash:    recorded.Hash,
				Address: recorded.Address,
				Title:   recorded.Title,
				Snippet: recorded.Snippet,
			},
			MatchedWords: wordCountsOf(recorded.MatchedWords),
		})
	}

	return answeredItems
}

func wordCountsOf(
	recordedWordCounts map[yacymodel.Hash]recordedWordCount,
) map[yacymodel.Hash]peeranswers.WordCount {
	wordCounts := make(map[yacymodel.Hash]peeranswers.WordCount, len(recordedWordCounts))
	for word, recorded := range recordedWordCounts {
		wordCounts[word] = peeranswers.WordCount{
			Hits:      recorded.Hits,
			TextWords: recorded.TextWords,
		}
	}

	return wordCounts
}

func recordedAnswersOf(query string, answers peeranswers.AnsweredQuery) recordedAnswers {
	itemsInTheOrderOfEachAnswer := make(
		[][]recordedItem, 0, len(answers.ItemsInTheOrderOfEachAnswer),
	)
	for _, itemsOfOneAnswer := range answers.ItemsInTheOrderOfEachAnswer {
		itemsInTheOrderOfEachAnswer = append(
			itemsInTheOrderOfEachAnswer, recordedItemsOf(itemsOfOneAnswer),
		)
	}

	return recordedAnswers{
		Query:                       query,
		RecordedAt:                  time.Now().UTC().Truncate(time.Second),
		ItemsInTheOrderOfEachAnswer: itemsInTheOrderOfEachAnswer,
		ItemsInNoOrder:              recordedItemsOf(answers.ItemsInNoOrder),
		DocumentsHeldPerQueryWord:   answers.DocumentsHeldPerQueryWord,
	}
}

func recordedItemsOf(answeredItems []peeranswers.AnsweredItem) []recordedItem {
	recordedItems := make([]recordedItem, 0, len(answeredItems))
	for _, answeredItem := range answeredItems {
		recordedItems = append(recordedItems, recordedItem{
			Hash:         answeredItem.Metadata.Hash,
			Address:      answeredItem.Metadata.Address,
			Title:        answeredItem.Metadata.Title,
			Snippet:      answeredItem.Metadata.Snippet,
			MatchedWords: recordedWordCountsOf(answeredItem.MatchedWords),
		})
	}

	return recordedItems
}

func recordedWordCountsOf(
	wordCounts map[yacymodel.Hash]peeranswers.WordCount,
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

	answersFiles, err := filepath.Glob(filepath.Join(recordedAnswersDirectory, "*.json"))
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

	content, err := os.ReadFile(path) //nolint:gosec // a fixture path of this test directory
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var answers recordedAnswers
	if err := json.Unmarshal(content, &answers); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return answers
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
