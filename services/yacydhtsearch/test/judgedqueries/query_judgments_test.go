package judgedqueries_test

import (
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
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
	Spam    bool              `json:"spam,omitempty"`
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
			spam:  judged.Spam,
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
	pageContentsPerDocument map[yacymodel.URLHash]queryanswers.PageContents,
	judgedAlready queryJudgments,
) queryJudgments {
	judgedDocumentPerHash := judgedAlready.judgedDocumentPerHash()

	documentsToJudge := documentsToJudgeOf(answers, pageContentsPerDocument, judgedDocumentPerHash)
	judgedDocuments := make([]judgedDocument, 0, len(documentsToJudge))
	for _, documentToJudge := range documentsToJudge {
		documentToJudge.Grade = judgedDocumentPerHash[documentToJudge.Hash].Grade
		documentToJudge.Spam = judgedDocumentPerHash[documentToJudge.Hash].Spam
		judgedDocuments = append(judgedDocuments, documentToJudge)
	}

	return queryJudgments{Query: query, JudgedDocuments: judgedDocuments}
}

func documentsToJudgeOf(
	answers queryanswers.AnsweredQuery,
	pageContentsPerDocument map[yacymodel.URLHash]queryanswers.PageContents,
	judgedDocumentPerHash map[yacymodel.URLHash]judgedDocument,
) []judgedDocument {
	toJudge := documentsAmongTheFirstOf(answers.FoundDocuments)
	maps.Copy(toJudge, documentsAmongTheFirstOf(
		orderingOfTheServiceFrom(
			documentrelevance.DefaultScoreWeights(),
		).OrderedDocumentsOf(answers),
	))
	for document := range pageContentsPerDocument {
		toJudge[document] = struct{}{}
	}
	for document, judged := range judgedDocumentPerHash {
		if judged.Grade == nil {
			continue
		}
		toJudge[document] = struct{}{}
	}

	documentsToJudge := make([]judgedDocument, 0, len(toJudge))
	for _, foundDocument := range answers.FoundDocuments {
		if _, judged := toJudge[foundDocument.Hash]; !judged {
			continue
		}
		documentsToJudge = append(documentsToJudge, judgedDocument{
			Hash:    foundDocument.Hash,
			Address: foundDocument.Address,
			Title:   foundDocument.Title,
		})
	}

	return documentsToJudge
}

func documentsAmongTheFirstOf(
	orderedDocuments []queryanswers.FoundDocument,
) map[yacymodel.URLHash]struct{} {
	documentsAmongTheFirst := make(map[yacymodel.URLHash]struct{}, judgedDocumentsCeiling)
	for _, orderedDocument := range orderedDocuments[:min(judgedDocumentsCeiling, len(orderedDocuments))] {
		documentsAmongTheFirst[orderedDocument.Hash] = struct{}{}
	}

	return documentsAmongTheFirst
}

func (j queryJudgments) judgedDocumentPerHash() map[yacymodel.URLHash]judgedDocument {
	judgedDocumentPerHash := make(map[yacymodel.URLHash]judgedDocument, len(j.JudgedDocuments))
	for _, judged := range j.JudgedDocuments {
		judgedDocumentPerHash[judged.Hash] = judged
	}

	return judgedDocumentPerHash
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
