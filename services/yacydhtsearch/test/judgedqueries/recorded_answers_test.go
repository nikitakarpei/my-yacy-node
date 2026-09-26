package judgedqueries_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	recordedAnswersDirectory  = "testdata/answers"
	recordedAnswersFileSuffix = ".json.gz"
)

type recordedAnswers struct {
	Query                     string                  `json:"query"`
	RecordedAt                time.Time               `json:"recordedAt"`
	FoundDocuments            []recordedFoundDocument `json:"foundDocuments"`
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int  `json:"documentsHeldPerQueryWord"`
}

type recordedFoundDocument struct {
	Hash         yacymodel.URLHash                         `json:"hash"`
	Metadata     yacymodel.Optional[yacymodel.URLMetadata] `json:"metadata,omitempty"`
	Postings     []recordedPostingReplica                  `json:"postings,omitempty"`
	PageContents yacymodel.Optional[recordedPageContents]  `json:"pageContents,omitempty"`
}

func recordedAnswersOf(
	query string, answersAndPageContents answersAndPageContents,
) recordedAnswers {
	return recordedAnswers{
		Query:                     query,
		RecordedAt:                time.Now().UTC().Truncate(time.Second),
		FoundDocuments:            recordedFoundDocumentsFrom(answersAndPageContents),
		DocumentsHeldPerQueryWord: answersAndPageContents.answers.DocumentsHeldPerQueryWord,
	}
}

func recordedFoundDocumentsFrom(
	answersAndPageContents answersAndPageContents,
) []recordedFoundDocument {
	answers := answersAndPageContents.answers
	recordedFoundDocuments := make([]recordedFoundDocument, 0, len(answers.FoundDocuments))
	for _, foundDocument := range answers.FoundDocuments {
		recordedFoundDocuments = append(recordedFoundDocuments, recordedFoundDocument{
			Hash:     foundDocument.Hash,
			Metadata: recordedMetadataOf(foundDocument),
			Postings: recordedPostingsOf(foundDocument),
			PageContents: recordedPageContentsFor(
				foundDocument.Hash, answersAndPageContents.pageContentsPerDocument,
			),
		})
	}

	return recordedFoundDocuments
}

func recordedMetadataOf(
	foundDocument queryanswers.FoundDocument,
) yacymodel.Optional[yacymodel.URLMetadata] {
	if len(foundDocument.MetadataReplicas) == 0 {
		return yacymodel.None[yacymodel.URLMetadata]()
	}

	return yacymodel.Some(foundDocument.MetadataReplicas[0].Metadata)
}

func (recorded recordedAnswers) withPageContentsReadAgain(
	answersAndPageContents answersAndPageContents,
) recordedAnswers {
	readAgain := recordedAnswersOf(recorded.Query, answersAndPageContents)
	readAgain.RecordedAt = recorded.RecordedAt

	return readAgain
}

func (recorded recordedAnswers) answers() queryanswers.AnsweredQuery {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(recorded.FoundDocuments))
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, recordedDocument := range recorded.FoundDocuments {
		foundDocuments = append(foundDocuments, queryanswers.FoundDocumentOf(
			recordedDocument.Hash,
			recordedDocument.metadataReplicas(),
			recordedDocument.postingReplicas(),
		))
		if pageContents, read := recordedDocument.PageContents.Get(); read {
			pageContentsPerDocument[recordedDocument.Hash] = pageContents.pageContents()
		}
	}

	query := queryreading.QueryFrom(recorded.Query, "")

	return queryanswers.AnsweredQuery{
		QueryWords:                query.WordHashes(),
		CompoundWords:             query.CompoundWords,
		FoundDocuments:            foundDocuments,
		DocumentsHeldPerQueryWord: recorded.DocumentsHeldPerQueryWord,
	}.WithReadPages(pageContentsPerDocument)
}

func (recorded recordedFoundDocument) metadataReplicas() []queryanswers.MetadataReplica {
	metadata, reported := recorded.Metadata.Get()
	if !reported {
		return nil
	}

	return []queryanswers.MetadataReplica{{Metadata: metadata}}
}

func (recorded recordedFoundDocument) postingReplicas() []queryanswers.PostingReplica {
	postingReplicas := make([]queryanswers.PostingReplica, 0, len(recorded.Postings))
	for _, posting := range recorded.Postings {
		postingReplicas = append(postingReplicas, queryanswers.PostingReplica{
			Holder:  posting.Holder,
			Word:    posting.Word,
			Posting: posting.Posting.posting(),
		})
	}

	return postingReplicas
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

func recordedAnswersFileOf(query string) string {
	return filepath.Join(
		recordedAnswersDirectory,
		queryInFileNames(query)+recordedAnswersFileSuffix,
	)
}

func recordedAnswersAt(t *testing.T, path string) recordedAnswers {
	t.Helper()

	var recorded recordedAnswers
	if err := json.Unmarshal(contentOfGzippedFixtureFile(t, path), &recorded); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return recorded
}

func writeRecordedAnswersFile(t *testing.T, path string, recorded recordedAnswers) {
	t.Helper()

	writeGzippedFixtureFile(t, path, append(indentedJSONOf(t, recorded), '\n'))
}

func answersWrittenAndReadBack(
	t *testing.T, answersAndPageContents answersAndPageContents,
) queryanswers.AnsweredQuery {
	t.Helper()

	path := filepath.Join(t.TempDir(), "recorded"+recordedAnswersFileSuffix)
	writeRecordedAnswersFile(t, path, recordedAnswersOf("berlin", answersAndPageContents))

	return recordedAnswersAt(t, path).answers()
}
