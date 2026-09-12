package judgedqueries_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
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
	answers peeranswers.AnsweredQuery,
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
	answers peeranswers.AnsweredQuery,
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
	answers peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]struct{} {
	orderedItems := peerorder.Ordering{}.OrderedItemsOf(answers)

	documentsPutFirst := make(map[yacymodel.URLHash]struct{}, judgedItemsCeiling)
	for _, orderedItem := range orderedItems[:min(judgedItemsCeiling, len(orderedItems))] {
		documentsPutFirst[orderedItem.Metadata.Hash] = struct{}{}
	}

	return documentsPutFirst
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
