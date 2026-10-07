package judgedqueries_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	evidenceImportSwitch = "YACYDHTSEARCH_IMPORT_JUDGED_QUERY_EVIDENCE"
	gradingDumpSwitch    = "YACYDHTSEARCH_DUMP_UNGRADED_JUDGED_DOCUMENTS"

	evidenceKeySeparator = "__"
)

type importedEvidence struct {
	Key         string    `json:"key"`
	Address     string    `json:"address"`
	ContentType string    `json:"contentType"`
	Body        []byte    `json:"body"`
	CapturedAt  time.Time `json:"capturedAt"`
	CapturedBy  string    `json:"capturedBy"`
	Status      int       `json:"status"`
}

func TestImportTheEvidencePagesOfTheJudgedQueries(t *testing.T) {
	evidenceDirectory := os.Getenv(evidenceImportSwitch)
	if evidenceDirectory == "" {
		t.Skipf("set %s to a directory of evidence files to import", evidenceImportSwitch)
	}

	evidencePerSlug := importedEvidencePerSlugIn(t, evidenceDirectory)
	for _, findingsFile := range recordedFindingsFiles(t) {
		recorded := recordedFindingsAt(t, findingsFile)
		if evidence := evidencePerSlug[queryInFileNames(recorded.Query)]; len(evidence) > 0 {
			importEvidenceOf(t, recorded.Query, evidence)
		}
	}
}

func importedEvidencePerSlugIn(
	t *testing.T,
	evidenceDirectory string,
) map[string][]importedEvidence {
	t.Helper()

	evidenceFiles, err := filepath.Glob(filepath.Join(evidenceDirectory, "*.json"))
	if err != nil {
		t.Fatalf("read %s: %v", evidenceDirectory, err)
	}
	evidencePerSlug := map[string][]importedEvidence{}
	for _, evidenceFile := range evidenceFiles {
		content, err := os.ReadFile(evidenceFile) //nolint:gosec // the operator's directory
		if err != nil {
			t.Fatalf("read %s: %v", evidenceFile, err)
		}
		var evidence importedEvidence
		if err := json.Unmarshal(content, &evidence); err != nil {
			t.Fatalf("read %s: %v", evidenceFile, err)
		}
		slug, _, separated := strings.Cut(evidence.Key, evidenceKeySeparator)
		if !separated {
			t.Fatalf("read %s: the key %q names no query and document", evidenceFile, evidence.Key)
		}
		evidencePerSlug[slug] = append(evidencePerSlug[slug], evidence)
	}

	return evidencePerSlug
}

func importEvidenceOf(t *testing.T, query string, evidence []importedEvidence) {
	t.Helper()

	alreadyHeld := evidencePagePerAddressOf(t, query)
	evidencePages := make([]storedPage, 0, len(evidence))
	gradePerDocument := map[yacymodel.URLHash]yacymodel.Optional[int]{}
	gonePerDocument := map[yacymodel.URLHash]captureFailure{}
	for _, imported := range evidence {
		_, hash, _ := strings.Cut(imported.Key, evidenceKeySeparator)
		document, err := yacymodel.ParseURLHash(hash)
		if err != nil {
			t.Fatalf("import the evidence of %q: %v", query, err)
		}
		if imported.Status == http.StatusNotFound || imported.Status == http.StatusGone {
			gonePerDocument[document] = goneCaptureFailure
			gradePerDocument[document] = yacymodel.Some(gradeOfAGonePage)
		}
		if _, held := alreadyHeld[imported.Address]; held {
			continue
		}
		if _, gone := gonePerDocument[document]; !gone {
			gradePerDocument[document] = yacymodel.None[int]()
		}
		evidencePages = append(evidencePages, storedPage{
			address:     imported.Address,
			contentType: imported.ContentType,
			body:        imported.Body,
			capturedAt:  imported.CapturedAt,
			capturedBy:  imported.CapturedBy,
			status:      imported.Status,
		})
	}
	if len(evidencePages) > 0 {
		appendEvidencePagesOf(t, query, evidencePages)
	}
	if len(gonePerDocument) > 0 {
		findingsFile := recordedFindingsFileOf(query)
		writeRecordedFindingsFile(t, findingsFile,
			recordedFindingsAt(t, findingsFile).withCaptureFailures(gonePerDocument))
	}
	t.Logf(
		"%q holds %d more evidence pages, %d of them of gone pages, and changes %d grades",
		query,
		len(evidencePages),
		len(gonePerDocument),
		regradedDocumentsOf(t, query, gradePerDocument),
	)
}

type documentToGrade struct {
	Query       string `json:"query"`
	Language    string `json:"language,omitempty"`
	Hash        string `json:"hash"`
	Address     string `json:"address"`
	Title       string `json:"title"`
	PeerSnippet string `json:"peerSnippet,omitempty"`
	Text        string `json:"text,omitempty"`
	TextSource  string `json:"textSource,omitempty"`
	CapturedAt  string `json:"capturedAt,omitempty"`
}

func TestDumpTheUngradedJudgedDocuments(t *testing.T) {
	dumpPath := os.Getenv(gradingDumpSwitch)
	if dumpPath == "" {
		t.Skipf("set %s to the file to write the ungraded judged documents to", gradingDumpSwitch)
	}

	extraction := pageExtractionOfEveryFormat(t)
	var lines []string
	for _, findingsFile := range recordedFindingsFiles(t) {
		for _, document := range extraction.documentsToGradeOf(t, recordedFindingsAt(t, findingsFile)) {
			line, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("write %s: %v", dumpPath, err)
			}
			lines = append(lines, string(line))
		}
	}
	if err := os.WriteFile( //nolint:gosec // the operator names the dump file
		dumpPath,
		[]byte(strings.Join(lines, "\n")+"\n"),
		fixtureFilePermissions,
	); err != nil {
		t.Fatalf("write %s: %v", dumpPath, err)
	}
	t.Logf("%d judged documents wait for a grade", len(lines))
}

func (extraction pageExtraction) documentsToGradeOf(
	t *testing.T, recorded recordedFindings,
) []documentToGrade {
	t.Helper()

	ungraded := map[yacymodel.URLHash]judgedDocument{}
	for _, judged := range queryJudgmentsAt(t, queryJudgmentsFileOf(recorded.Query)).JudgedDocuments {
		if !judged.Grade.Present() {
			ungraded[judged.Hash] = judged
		}
	}
	storedPagePerAddress := storedPagePerAddressOf(t, recorded.Query)
	evidencePagePerAddress := evidencePagePerAddressOf(t, recorded.Query)
	var documents []documentToGrade
	for _, foundDocument := range recorded.FoundDocuments {
		judged, waiting := ungraded[foundDocument.Hash]
		if !waiting || foundDocument.CaptureFailure == passingCaptureFailure ||
			foundDocument.CaptureFailure == goneCaptureFailure {
			continue
		}
		document := documentToGrade{
			Query: recorded.Query, Language: recorded.Language.String(),
			Hash: judged.Hash.String(), Address: judged.Address, Title: judged.Title,
		}
		metadata, reported := foundDocument.Metadata.Get()
		if reported {
			document.PeerSnippet = metadata.Snippet
			document = extraction.withTextOf(
				t,
				document,
				storedPagePerAddress[metadata.Address],
				evidencePagePerAddress[metadata.Address],
				foundDocument.CaptureFailure,
			)
		}
		documents = append(documents, document)
	}

	return documents
}

func (extraction pageExtraction) withTextOf(
	t *testing.T,
	document documentToGrade,
	storedPage, evidencePage storedPage,
	failure captureFailure,
) documentToGrade {
	t.Helper()

	if failure != refusedCaptureFailure {
		if withText, held := extraction.withTextFrom(
			t,
			document,
			storedPage,
			"stored",
		); held {
			return withText
		}
	}
	if withText, held := extraction.withTextFrom(
		t,
		document,
		evidencePage,
		"evidence",
	); held {
		return withText
	}

	return document
}

func (extraction pageExtraction) withTextFrom(
	t *testing.T, document documentToGrade, page storedPage, textSource string,
) (documentToGrade, bool) {
	t.Helper()

	if len(page.body) == 0 || statusOf(page) >= http.StatusBadRequest {
		return document, false
	}
	text := strings.Join(strings.Fields(extraction.extractedPageOf(t.Context(), page).text), " ")
	if text == "" {
		return document, false
	}
	document.Text, document.TextSource = text, textSource
	if !page.capturedAt.IsZero() {
		document.CapturedAt = page.capturedAt.Format(time.RFC3339)
	}

	return document, true
}
