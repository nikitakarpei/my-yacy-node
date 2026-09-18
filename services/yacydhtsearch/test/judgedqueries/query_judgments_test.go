package judgedqueries_test

import (
	"encoding/json"
	"errors"
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
	Hash     yacymodel.URLHash `json:"hash"`
	Address  string            `json:"address"`
	Title    string            `json:"title"`
	Grade    *int              `json:"grade"`
	Subtopic string            `json:"subtopic,omitempty"`
}

func (j queryJudgments) gradedDocumentsOfTheQuery() gradedDocuments {
	graded := make(gradedDocuments, len(j.JudgedDocuments))
	for _, judged := range j.JudgedDocuments {
		if judged.Grade == nil {
			continue
		}
		graded[judged.Hash] = gradedDocument{
			grade:          *judged.Grade,
			hash:           judged.Hash,
			judgedSubtopic: judged.Subtopic,
		}
	}

	return graded
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
	judgmentPerDocument := judgedAlready.judgmentPerJudgedDocument()

	documentsToJudge := documentsToJudgeOf(answers, pageTextPerDocument)
	judgedDocuments := make([]judgedDocument, 0, len(documentsToJudge))
	for _, documentToJudge := range documentsToJudge {
		documentToJudge.Grade = judgmentPerDocument[documentToJudge.Hash].Grade
		documentToJudge.Subtopic = judgmentPerDocument[documentToJudge.Hash].Subtopic
		judgedDocuments = append(judgedDocuments, documentToJudge)
	}

	return queryJudgments{Query: query, JudgedDocuments: judgedDocuments}
}

func documentsToJudgeOf(
	answers queryanswers.AnsweredQuery,
	pageTextPerDocument map[yacymodel.URLHash]string,
) []judgedDocument {
	toJudge := documentsThePeersPutFirst(answers)
	for document := range pageTextPerDocument {
		toJudge[document] = struct{}{}
	}

	answeredItems := answers.ItemOfEachAnsweredDocument()
	documentsToJudge := make([]judgedDocument, 0, len(toJudge))
	for _, answeredItem := range answeredItems {
		if _, judged := toJudge[answeredItem.Metadata.Hash]; !judged {
			continue
		}
		documentsToJudge = append(documentsToJudge, judgedDocument{
			Hash:    answeredItem.Metadata.Hash,
			Address: answeredItem.Metadata.Address,
			Title:   answeredItem.Metadata.Title,
		})
	}

	return documentsToJudge
}

func documentsThePeersPutFirst(
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]struct{} {
	orderedItems := answers.ItemOfEachAnsweredDocument()

	documentsPutFirst := make(map[yacymodel.URLHash]struct{}, judgedItemsCeiling)
	for _, orderedItem := range orderedItems[:min(judgedItemsCeiling, len(orderedItems))] {
		documentsPutFirst[orderedItem.Metadata.Hash] = struct{}{}
	}

	return documentsPutFirst
}

func (j queryJudgments) judgmentPerJudgedDocument() map[yacymodel.URLHash]judgedDocument {
	judgmentPerDocument := make(map[yacymodel.URLHash]judgedDocument, len(j.JudgedDocuments))
	for _, judged := range j.JudgedDocuments {
		judgmentPerDocument[judged.Hash] = judged
	}

	return judgmentPerDocument
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
