package judgedqueries_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
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
	Hash    yacymodel.URLHash `json:"hash"`
	Address string            `json:"address"`
	Title   string            `json:"title"`
	Grade   *int              `json:"grade"`
}

func (j queryJudgments) gradedDocumentsOfTheQuery() gradedDocuments {
	graded := make(gradedDocuments, len(j.JudgedDocuments))
	for _, judged := range j.JudgedDocuments {
		if judged.Grade == nil {
			continue
		}
		graded[judged.Hash] = gradedDocument{
			grade: *judged.Grade,
			host:  hostOf(judged.Address),
		}
	}

	return graded
}

func hostOf(address string) string {
	readAddress, err := url.Parse(address)
	if err != nil || readAddress.Hostname() == "" {
		return address
	}

	return readAddress.Hostname()
}

func (j queryJudgments) amountOfUngradedDocuments() int {
	amountOfUngradedDocuments := 0
	for _, judged := range j.JudgedDocuments {
		if judged.Grade != nil {
			continue
		}
		amountOfUngradedDocuments++
	}

	return amountOfUngradedDocuments
}

func queryJudgmentsOfTheDocumentsToJudge(
	query string,
	answers queryanswers.AnsweredQuery,
	pageTextPerDocument map[yacymodel.URLHash]string,
	judgedAlready queryJudgments,
) queryJudgments {
	gradePerDocument := judgedAlready.gradePerJudgedDocument()

	documentsToJudge := documentsToJudgeOf(answers, pageTextPerDocument)
	judgedDocuments := make([]judgedDocument, 0, len(documentsToJudge))
	for _, documentToJudge := range documentsToJudge {
		documentToJudge.Grade = gradePerDocument[documentToJudge.Hash]
		judgedDocuments = append(judgedDocuments, documentToJudge)
	}

	return queryJudgments{Query: query, JudgedDocuments: judgedDocuments}
}

func documentsToJudgeOf(
	answers queryanswers.AnsweredQuery,
	pageTextPerDocument map[yacymodel.URLHash]string,
) []judgedDocument {
	toJudge := documentsFoundFirst(answers)
	for document := range pageTextPerDocument {
		toJudge[document] = struct{}{}
	}

	documentsToJudge := make([]judgedDocument, 0, len(toJudge))
	for _, foundDocument := range answers.FoundDocuments {
		if _, judged := toJudge[foundDocument.Metadata.Hash]; !judged {
			continue
		}
		documentsToJudge = append(documentsToJudge, judgedDocument{
			Hash:    foundDocument.Metadata.Hash,
			Address: foundDocument.Metadata.Address,
			Title:   foundDocument.Metadata.Title,
		})
	}

	return documentsToJudge
}

func documentsFoundFirst(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]struct{} {
	foundDocuments := answers.FoundDocuments

	documentsAmongTheFirstFound := make(map[yacymodel.URLHash]struct{}, judgedItemsCeiling)
	for _, foundDocument := range foundDocuments[:min(judgedItemsCeiling, len(foundDocuments))] {
		documentsAmongTheFirstFound[foundDocument.Metadata.Hash] = struct{}{}
	}

	return documentsAmongTheFirstFound
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
