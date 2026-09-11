package judgedqueries_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const queryJudgmentsDirectory = "testdata/judgments"

type queryJudgments struct {
	Query           string           `json:"query"`
	JudgedDocuments []judgedDocument `json:"judgedDocuments"`
}

type judgedDocument struct {
	Hash    yacymodel.URLHash `json:"hash"`
	Address string            `json:"address"`
	Title   string            `json:"title"`
	Grade   *int              `json:"grade"`
}

func (j queryJudgments) gradeOfEachDocument() gradedDocuments {
	grades := make(gradedDocuments, len(j.JudgedDocuments))
	for _, judged := range j.JudgedDocuments {
		grades[judged.Hash] = judged.gradeWhenJudged()
	}

	return grades
}

func (d judgedDocument) gradeWhenJudged() int {
	if d.Grade == nil {
		return 0
	}

	return *d.Grade
}

func queryJudgmentsOfThePool(
	query string,
	pooledDocuments []judgedDocument,
	judgedAlready queryJudgments,
) queryJudgments {
	gradePerDocument := judgedAlready.gradePerJudgedDocument()

	judgedDocuments := make([]judgedDocument, 0, len(pooledDocuments))
	for _, pooled := range pooledDocuments {
		pooled.Grade = gradePerDocument[pooled.Hash]
		judgedDocuments = append(judgedDocuments, pooled)
	}

	return queryJudgments{Query: query, JudgedDocuments: judgedDocuments}
}

func (j queryJudgments) gradePerJudgedDocument() map[yacymodel.URLHash]*int {
	gradePerDocument := make(map[yacymodel.URLHash]*int, len(j.JudgedDocuments))
	for _, judged := range j.JudgedDocuments {
		gradePerDocument[judged.Hash] = judged.Grade
	}

	return gradePerDocument
}

func queryJudgmentsInTheFile(t *testing.T, path string) queryJudgments {
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
