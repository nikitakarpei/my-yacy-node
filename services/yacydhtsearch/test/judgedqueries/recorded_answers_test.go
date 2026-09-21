package judgedqueries_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
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
	Hash         yacymodel.URLHash                         `json:"hash"`
	Metadata     yacymodel.Optional[yacymodel.URLMetadata] `json:"metadata,omitempty"`
	Postings     []recordedPostingReplica                  `json:"postings,omitempty"`
	PageContents yacymodel.Optional[recordedPageContents]  `json:"pageContents,omitempty"`
}

type recordedPageContents struct {
	Address          string                 `json:"address,omitempty"`
	Title            string                 `json:"title"`
	Snippet          string                 `json:"snippet"`
	HitsPerQueryWord map[yacymodel.Hash]int `json:"hitsPerQueryWord"`
	QueryPhraseHits  int                    `json:"queryPhraseHits"`
	AmountOfWords    int                    `json:"amountOfWords"`
	LocalLinks       int                    `json:"localLinks"`
	ExternalLinks    int                    `json:"externalLinks"`
}

func recordedPageContentsOf(pageContents pagecontents.PageContents) recordedPageContents {
	return recordedPageContents{
		Address:          pageContents.Address,
		Title:            pageContents.Title,
		Snippet:          pageContents.Snippet,
		HitsPerQueryWord: pageContents.HitsPerQueryWord,
		QueryPhraseHits:  pageContents.QueryPhraseHits,
		AmountOfWords:    pageContents.AmountOfWords,
		LocalLinks:       pageContents.LinkCounts.LocalLinks,
		ExternalLinks:    pageContents.LinkCounts.ExternalLinks,
	}
}

func (r recordedPageContents) pageContents() pagecontents.PageContents {
	return pagecontents.PageContents{
		Address:          r.Address,
		Title:            r.Title,
		Snippet:          r.Snippet,
		HitsPerQueryWord: r.HitsPerQueryWord,
		QueryPhraseHits:  r.QueryPhraseHits,
		AmountOfWords:    r.AmountOfWords,
		LinkCounts: pagecontents.LinkCounts{
			LocalLinks:    r.LocalLinks,
			ExternalLinks: r.ExternalLinks,
		},
	}
}

type recordedPostingReplica struct {
	Holder  yacymodel.Hash                     `json:"holder"`
	Word    yacymodel.Optional[yacymodel.Hash] `json:"word,omitempty"`
	Posting recordedPosting                    `json:"posting"`
}

type recordedPosting struct {
	yacymodel.RWIPosting

	WordHash *yacymodel.Hash `json:"WordHash,omitempty"`
}

func recordedPostingOf(posting yacymodel.RWIPosting) recordedPosting {
	recorded := recordedPosting{RWIPosting: posting}
	if !posting.WordHash.IsZero() {
		wordHash := posting.WordHash
		recorded.WordHash = &wordHash
	}
	recorded.RWIPosting.WordHash = yacymodel.Hash{}

	return recorded
}

func (r recordedPosting) posting() yacymodel.RWIPosting {
	posting := r.RWIPosting
	if r.WordHash != nil {
		posting.WordHash = *r.WordHash
	}

	return posting
}

func (r recordedAnswers) answeredQuery() queryanswers.AnsweredQuery {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(r.FoundDocuments))
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, recorded := range r.FoundDocuments {
		foundDocuments = append(foundDocuments, queryanswers.FoundDocumentOf(
			recorded.Hash, recorded.metadataReplicas(), recorded.postingReplicas(),
		))
		if pageContentsRecorded, read := recorded.PageContents.Get(); read {
			pageContentsPerDocument[recorded.Hash] = pageContentsRecorded.pageContents()
		}
	}

	return queryanswers.AnsweredQuery{
		QueryWords:                searchquery.QueryFrom(r.Query, "").TermHashes(),
		FoundDocuments:            foundDocuments,
		DocumentsHeldPerQueryWord: r.DocumentsHeldPerQueryWord,
	}.WithReadPages(pageContentsPerDocument)
}

func (r recordedFoundDocument) metadataReplicas() []queryanswers.MetadataReplica {
	metadata, reported := r.Metadata.Get()
	if !reported {
		return nil
	}

	return []queryanswers.MetadataReplica{{Metadata: metadata}}
}

func (r recordedFoundDocument) postingReplicas() []queryanswers.PostingReplica {
	postingReplicas := make([]queryanswers.PostingReplica, 0, len(r.Postings))
	for _, posting := range r.Postings {
		postingReplicas = append(postingReplicas, queryanswers.PostingReplica{
			Holder:  posting.Holder,
			Word:    posting.Word,
			Posting: posting.Posting.posting(),
		})
	}

	return postingReplicas
}

func recordedAnswersOf(
	query string, answersAndTheirPages answersAndTheirReadPages,
) recordedAnswers {
	return recordedAnswers{
		Query:                     query,
		RecordedAt:                time.Now().UTC().Truncate(time.Second),
		FoundDocuments:            recordedFoundDocumentsFrom(answersAndTheirPages),
		DocumentsHeldPerQueryWord: answersAndTheirPages.answeredQuery.DocumentsHeldPerQueryWord,
	}
}

func (r recordedAnswers) withThePageContentsReadAgain(
	answersAndTheirPages answersAndTheirReadPages,
) recordedAnswers {
	readAgain := recordedAnswersOf(r.Query, answersAndTheirPages)
	readAgain.RecordedAt = r.RecordedAt

	return readAgain
}

func recordedFoundDocumentsFrom(
	answersAndTheirPages answersAndTheirReadPages,
) []recordedFoundDocument {
	answers := answersAndTheirPages.answeredQuery
	recordedFoundDocuments := make([]recordedFoundDocument, 0, len(answers.FoundDocuments))
	for _, foundDocument := range answers.FoundDocuments {
		recordedFoundDocuments = append(recordedFoundDocuments, recordedFoundDocument{
			Hash:     foundDocument.Hash,
			Metadata: metadataRecordedOf(foundDocument),
			Postings: postingsRecordedOf(foundDocument),
			PageContents: pageContentsRecordedFor(
				foundDocument.Hash, answersAndTheirPages.pageContentsPerDocument,
			),
		})
	}

	return recordedFoundDocuments
}

func metadataRecordedOf(
	foundDocument queryanswers.FoundDocument,
) yacymodel.Optional[yacymodel.URLMetadata] {
	if len(foundDocument.MetadataReplicas) == 0 {
		return yacymodel.None[yacymodel.URLMetadata]()
	}

	return yacymodel.Some(foundDocument.MetadataReplicas[0].Metadata)
}

func postingsRecordedOf(
	foundDocument queryanswers.FoundDocument,
) []recordedPostingReplica {
	postings := make([]recordedPostingReplica, 0, len(foundDocument.PostingReplicas))
	for _, replica := range foundDocument.PostingReplicas {
		postings = append(postings, recordedPostingReplica{
			Holder:  replica.Holder,
			Word:    replica.Word,
			Posting: recordedPostingOf(replica.Posting),
		})
	}

	return postings
}

func pageContentsRecordedFor(
	document yacymodel.URLHash,
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) yacymodel.Optional[recordedPageContents] {
	pageContents, read := pageContentsPerDocument[document]
	if !read {
		return yacymodel.None[recordedPageContents]()
	}

	return yacymodel.Some(recordedPageContentsOf(pageContents))
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
	if err := json.Unmarshal(contentOfTheGzippedFixtureFile(t, path), &answers); err != nil {
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
	writeGzippedFixtureFile(t, path, append(content, '\n'))
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
