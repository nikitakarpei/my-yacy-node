package judgedqueries_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	recordedFindingsDirectory  = "testdata/findings"
	recordedFindingsFileSuffix = ".json.gz"
)

type recordedFindings struct {
	Query                     string                  `json:"query"`
	Language                  yacymodel.Language      `json:"language,omitzero"`
	RecordedAt                time.Time               `json:"recordedAt"`
	FoundDocuments            []recordedFoundDocument `json:"foundDocuments"`
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int  `json:"documentsHeldPerQueryWord"`
}

type recordedFoundDocument struct {
	Hash           yacymodel.URLHash                         `json:"hash"`
	Metadata       yacymodel.Optional[yacymodel.URLMetadata] `json:"metadata,omitempty"`
	Postings       []recordedPostingReplica                  `json:"postings,omitempty"`
	PageContents   yacymodel.Optional[recordedPageContents]  `json:"pageContents,omitempty"`
	CaptureFailure captureFailure                            `json:"captureFailure,omitempty"`
}

func recordedFindingsOf(
	query string, language yacymodel.Language, findingsAndPageContents findingsAndPageContents,
) recordedFindings {
	return recordedFindings{
		Query:                     query,
		Language:                  language,
		RecordedAt:                time.Now().UTC().Truncate(time.Second),
		FoundDocuments:            recordedFoundDocumentsFrom(findingsAndPageContents),
		DocumentsHeldPerQueryWord: findingsAndPageContents.findings.DocumentsHeldPerQueryWord,
	}
}

func recordedFoundDocumentsFrom(
	findingsAndPageContents findingsAndPageContents,
) []recordedFoundDocument {
	findings := findingsAndPageContents.findings
	recordedFoundDocuments := make([]recordedFoundDocument, 0, len(findings.FoundDocuments))
	for _, foundDocument := range findings.FoundDocuments {
		recordedFoundDocuments = append(recordedFoundDocuments, recordedFoundDocument{
			Hash:     foundDocument.Hash,
			Metadata: recordedMetadataOf(foundDocument),
			Postings: recordedPostingsOf(foundDocument),
			PageContents: recordedPageContentsFor(
				foundDocument.Hash, findingsAndPageContents.pageContentsPerDocument,
			),
		})
	}

	return recordedFoundDocuments
}

func recordedMetadataOf(
	foundDocument queryfindings.FoundDocument,
) yacymodel.Optional[yacymodel.URLMetadata] {
	if len(foundDocument.MetadataReplicas) == 0 {
		return yacymodel.None[yacymodel.URLMetadata]()
	}

	return yacymodel.Some(foundDocument.MetadataReplicas[0].Metadata)
}

func (recorded recordedFindings) withPageContentsReadAgain(
	findingsAndPageContents findingsAndPageContents,
) recordedFindings {
	readAgain := recordedFindingsOf(recorded.Query, recorded.Language, findingsAndPageContents)
	readAgain.RecordedAt = recorded.RecordedAt

	return readAgain.withCaptureFailures(recorded.captureFailurePerDocument())
}

func (recorded recordedFindings) withCaptureFailures(
	captureFailurePerDocument map[yacymodel.URLHash]captureFailure,
) recordedFindings {
	foundDocuments := make([]recordedFoundDocument, 0, len(recorded.FoundDocuments))
	for _, foundDocument := range recorded.FoundDocuments {
		if failure, captured := captureFailurePerDocument[foundDocument.Hash]; captured {
			foundDocument.CaptureFailure = failure
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	recorded.FoundDocuments = foundDocuments

	return recorded
}

func (recorded recordedFindings) captureFailurePerDocument() map[yacymodel.URLHash]captureFailure {
	captureFailurePerDocument := map[yacymodel.URLHash]captureFailure{}
	for _, foundDocument := range recorded.FoundDocuments {
		if foundDocument.CaptureFailure != "" {
			captureFailurePerDocument[foundDocument.Hash] = foundDocument.CaptureFailure
		}
	}

	return captureFailurePerDocument
}

func (recorded recordedFindings) findings() queryfindings.Findings {
	foundDocuments := make([]queryfindings.FoundDocument, 0, len(recorded.FoundDocuments))
	for _, recordedDocument := range recorded.FoundDocuments {
		foundDocuments = append(foundDocuments, queryfindings.FoundDocumentOf(
			recordedDocument.Hash,
			recordedDocument.metadataReplicas(),
			recordedDocument.postingReplicas(),
		))
	}

	query := queryreading.QueryFrom(recorded.Query, recorded.Language)

	return queryfindings.Findings{
		QueryWords:                query.WordHashes(),
		CompoundWords:             query.CompoundWords,
		FoundDocuments:            foundDocuments,
		DocumentsHeldPerQueryWord: recorded.DocumentsHeldPerQueryWord,
	}.WithReadPages(recorded.pageContentsPerDocument())
}

func (recorded recordedFindings) pageContentsPerDocument() map[yacymodel.URLHash]pagecontents.PageContents {
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, recordedDocument := range recorded.FoundDocuments {
		if pageContents, read := recordedDocument.PageContents.Get(); read {
			pageContentsPerDocument[recordedDocument.Hash] = pageContents.pageContents()
		}
	}

	return pageContentsPerDocument
}

func (recorded recordedFoundDocument) metadataReplicas() []queryfindings.MetadataReplica {
	metadata, reported := recorded.Metadata.Get()
	if !reported {
		return nil
	}

	return []queryfindings.MetadataReplica{{Metadata: metadata}}
}

func (recorded recordedFoundDocument) postingReplicas() []queryfindings.PostingReplica {
	postingReplicas := make([]queryfindings.PostingReplica, 0, len(recorded.Postings))
	for _, posting := range recorded.Postings {
		postingReplicas = append(postingReplicas, queryfindings.PostingReplica{
			Holder:  posting.Holder,
			Word:    posting.Word,
			Posting: posting.Posting.posting(),
		})
	}

	return postingReplicas
}

func recordedFindingsFiles(t *testing.T) []string {
	t.Helper()

	findingsFiles, err := filepath.Glob(
		filepath.Join(recordedFindingsDirectory, "*"+recordedFindingsFileSuffix),
	)
	if err != nil {
		t.Fatalf("read %s: %v", recordedFindingsDirectory, err)
	}
	if len(findingsFiles) == 0 {
		t.Fatalf("no query is recorded in %s", recordedFindingsDirectory)
	}

	return findingsFiles
}

func recordedFindingsFileOf(query string) string {
	return filepath.Join(
		recordedFindingsDirectory,
		queryInFileNames(query)+recordedFindingsFileSuffix,
	)
}

func recordedFindingsAt(t *testing.T, path string) recordedFindings {
	t.Helper()

	var recorded recordedFindings
	if err := json.Unmarshal(contentOfGzippedFixtureFile(t, path), &recorded); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return recorded
}

func writeRecordedFindingsFile(t *testing.T, path string, recorded recordedFindings) {
	t.Helper()

	writeGzippedFixtureFile(t, path, append(indentedJSONOf(t, recorded), '\n'))
}

func findingsWrittenAndReadBack(
	t *testing.T, findingsAndPageContents findingsAndPageContents,
) queryfindings.Findings {
	t.Helper()

	path := filepath.Join(t.TempDir(), "recorded"+recordedFindingsFileSuffix)
	writeRecordedFindingsFile(
		t,
		path,
		recordedFindingsOf("berlin", yacymodel.Language{}, findingsAndPageContents),
	)

	return recordedFindingsAt(t, path).findings()
}
