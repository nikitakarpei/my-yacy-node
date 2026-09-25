package judgedqueries_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	queryJudgmentsDirectory  = "testdata/judgments"
	queryJudgmentsFileSuffix = ".json"
)

type queryJudgments struct {
	Query           string           `json:"query"`
	JudgedDocuments []judgedDocument `json:"judgedDocuments"`
}

type judgedDocument struct {
	Hash    yacymodel.URLHash       `json:"hash"`
	Address string                  `json:"address"`
	Title   string                  `json:"title"`
	Grade   yacymodel.Optional[int] `json:"grade"`
	Spam    bool                    `json:"spam,omitempty"`
}

func writeJudgmentsOf(
	t *testing.T, query string, answersAndPageContents answersAndPageContents,
) queryJudgments {
	t.Helper()

	judgments := queryJudgmentsOf(t, query).
		withDocumentsToJudgeIn(answersAndPageContents.answers)
	writeFixtureFile(t, queryJudgmentsFileOf(query), judgments)

	return judgments
}

func queryJudgmentsOf(t *testing.T, query string) queryJudgments {
	t.Helper()

	judgments := queryJudgmentsAt(t, queryJudgmentsFileOf(query))
	judgments.Query = query

	return judgments
}

func queryJudgmentsAt(t *testing.T, path string) queryJudgments {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // a fixture path of this test directory
	if errors.Is(err, os.ErrNotExist) {
		return queryJudgments{}
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var judgments queryJudgments
	if err := json.Unmarshal(content, &judgments); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return judgments
}

func queryJudgmentsFileOf(query string) string {
	return filepath.Join(queryJudgmentsDirectory, queryInFileNames(query)+queryJudgmentsFileSuffix)
}

func (judgments queryJudgments) withDocumentsToJudgeIn(
	answers queryanswers.AnsweredQuery,
) queryJudgments {
	judgedDocumentPerHash := judgments.judgedDocumentPerHash()

	documentsToJudge := judgments.documentsToJudgeIn(answers)
	judgedDocuments := make([]judgedDocument, 0, len(documentsToJudge))
	for _, documentToJudge := range documentsToJudge {
		documentToJudge.Grade = judgedDocumentPerHash[documentToJudge.Hash].Grade
		documentToJudge.Spam = judgedDocumentPerHash[documentToJudge.Hash].Spam
		judgedDocuments = append(judgedDocuments, documentToJudge)
	}

	return queryJudgments{Query: judgments.Query, JudgedDocuments: judgedDocuments}
}

func (judgments queryJudgments) judgedDocumentPerHash() map[yacymodel.URLHash]judgedDocument {
	judgedDocumentPerHash := make(
		map[yacymodel.URLHash]judgedDocument, len(judgments.JudgedDocuments),
	)
	for _, judged := range judgments.JudgedDocuments {
		judgedDocumentPerHash[judged.Hash] = judged
	}

	return judgedDocumentPerHash
}

func (judgments queryJudgments) documentsToJudgeIn(
	answers queryanswers.AnsweredQuery,
) []judgedDocument {
	documentsToJudge := hashesOf(theFirstOf(answers.FoundDocuments))
	maps.Copy(documentsToJudge, hashesOf(theFirstOf(
		defaultServiceOrdering().OrderedDocumentsOf(context.Background(), answers),
	)))
	for _, foundDocument := range answers.FoundDocuments {
		if !foundDocument.Facts.AmountOfWords.Present() {
			continue
		}
		documentsToJudge[foundDocument.Hash] = struct{}{}
	}
	for document, judged := range judgments.judgedDocumentPerHash() {
		if !judged.Grade.Present() {
			continue
		}
		documentsToJudge[document] = struct{}{}
	}

	judgedDocuments := make([]judgedDocument, 0, len(documentsToJudge))
	for _, foundDocument := range answers.FoundDocuments {
		if _, toJudge := documentsToJudge[foundDocument.Hash]; !toJudge {
			continue
		}
		judgedDocuments = append(judgedDocuments, judgedDocument{
			Hash:    foundDocument.Hash,
			Address: foundDocument.Address,
			Title:   foundDocument.Title,
		})
	}

	return judgedDocuments
}

func hashesOf(documents []queryanswers.FoundDocument) map[yacymodel.URLHash]struct{} {
	hashes := make(map[yacymodel.URLHash]struct{}, len(documents))
	for _, document := range documents {
		hashes[document.Hash] = struct{}{}
	}

	return hashes
}

func (judgments queryJudgments) gradedDocuments() gradedDocuments {
	gradedPerHash := make(gradedDocuments, len(judgments.JudgedDocuments))
	for _, judged := range judgments.JudgedDocuments {
		grade, graded := judged.Grade.Get()
		if !graded {
			continue
		}
		gradedPerHash[judged.Hash] = gradedDocument{
			grade: grade,
			site:  yacymodel.SiteOf(judged.Address),
			spam:  judged.Spam,
		}
	}

	return gradedPerHash
}

func (judgments queryJudgments) amountOfUngradedDocuments() int {
	amountOfUngradedDocuments := 0
	for _, judged := range judgments.JudgedDocuments {
		if judged.Grade.Present() {
			continue
		}
		amountOfUngradedDocuments++
	}

	return amountOfUngradedDocuments
}

func queryJudgmentsFiles(t *testing.T) []string {
	t.Helper()

	judgmentsFiles, err := filepath.Glob(
		filepath.Join(queryJudgmentsDirectory, "*"+queryJudgmentsFileSuffix),
	)
	if err != nil {
		t.Fatalf("read %s: %v", queryJudgmentsDirectory, err)
	}
	if len(judgmentsFiles) == 0 {
		t.Fatalf("no query is judged in %s", queryJudgmentsDirectory)
	}

	return judgmentsFiles
}
