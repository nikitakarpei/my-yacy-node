package queryfindings_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

func findingsOfReadAndUnreadDocument(t *testing.T) queryfindings.Findings {
	t.Helper()

	findings := findingsOfReadDocument(t)
	unread := findingsOfDocumentAPeerMatchedForWord(t, "https://unread.example/", "berlin")
	findings.FoundDocuments = append(unread.FoundDocuments, findings.FoundDocuments...)

	return findings
}

func TestOnlyReadPagesLeaveOutDocumentWhosePageWasNotRead(t *testing.T) {
	t.Parallel()

	read := findingsOfReadAndUnreadDocument(t).WithOnlyReadPages(pageContentsOfReadDocument(t))

	if len(read.FoundDocuments) != 1 || read.FoundDocuments[0].Address != addressOfReadDocument {
		t.Fatalf(
			"the findings hold %+v, want only the document whose page was read",
			read.FoundDocuments,
		)
	}
}

func TestOnlyReadPagesGiveReadDocumentWhatItsTextHolds(t *testing.T) {
	t.Parallel()

	read := findingsOfReadAndUnreadDocument(t).WithOnlyReadPages(pageContentsOfReadDocument(t))

	foundDocument := read.FoundDocuments[0]
	if foundDocument.Snippet != "Berlin holds a wall." || foundDocument.Title != "Berlin" ||
		foundDocument.Facts.AmountOfWords.OrElse(0) != 400 {
		t.Fatalf(
			"the document reads %+v, want the snippet, the title and the counts of the text",
			foundDocument,
		)
	}
}

func TestOnlyReadPagesLeaveNoDocumentWhenNoPageWasRead(t *testing.T) {
	t.Parallel()

	read := findingsOfReadAndUnreadDocument(t).WithOnlyReadPages(nil)

	if len(read.FoundDocuments) != 0 {
		t.Fatalf("the findings hold %+v, want no document", read.FoundDocuments)
	}
}
