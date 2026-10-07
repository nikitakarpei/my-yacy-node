package judgedqueries_test

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestThePagesOfAWARCReadBackWhatWasWrittenIntoIt(t *testing.T) {
	t.Parallel()

	writtenPages := []storedPage{
		{
			address:     "https://example.org/weather",
			contentType: "text/html; charset=utf-8",
			body:        []byte("<html><body>Rain in Berlin.\r\n\r\nMore rain.</body></html>"),
		},
		{
			address:     "https://example.org/report.pdf",
			contentType: "application/pdf",
			body:        []byte{0x25, 0x50, 0x44, 0x46, 0x00, 0x0d, 0x0a, 0xff},
		},
	}

	readPages := pagesOfWARC(t, warcOf(t, writtenPages))

	if len(readPages) != len(writtenPages) {
		t.Fatalf("the warc holds %d pages, want %d", len(readPages), len(writtenPages))
	}
	for place, page := range readPages {
		if page.address != writtenPages[place].address ||
			page.contentType != writtenPages[place].contentType ||
			!bytes.Equal(page.body, writtenPages[place].body) {
			t.Fatalf(
				"the warc holds the page %q with the content type %q and %d bytes, want "+
					"%q, %q and %d bytes",
				page.address,
				page.contentType,
				len(page.body),
				writtenPages[place].address,
				writtenPages[place].contentType,
				len(writtenPages[place].body),
			)
		}
	}
}

func TestPagesAppendedToAPagesFileReadBackAfterTheEarlierOnesThatStayAsTheyWere(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "query"+storedPagesFileSuffix)
	earlierPage := storedPage{
		address: "https://example.org/earlier", contentType: "text/html", body: []byte("earlier"),
	}
	appendedPage := storedPage{
		address: "https://example.org/appended", contentType: "text/html", body: []byte("appended"),
	}
	writeZstandardFixtureFile(t, path, warcOf(t, []storedPage{earlierPage}))
	earlierFile, err := os.ReadFile(path) //nolint:gosec // a path of the test's own directory
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	appendZstandardFrameTo(t, path, warcOf(t, []storedPage{appendedPage}))

	appendedFile, err := os.ReadFile(path) //nolint:gosec // a path of the test's own directory
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.HasPrefix(appendedFile, earlierFile) {
		t.Fatalf("the pages file no longer starts with the bytes it held before the append")
	}
	readPages := pagesOfWARC(t, contentOfZstandardFixtureFile(t, path))
	if len(readPages) != 2 || readPages[0].address != earlierPage.address ||
		readPages[1].address != appendedPage.address ||
		!bytes.Equal(readPages[1].body, appendedPage.body) {
		t.Fatalf("the pages file holds %+v, want the earlier page and then the appended one",
			readPages)
	}
}

func TestAPageOfAWARCReadsBackWhenAndByWhatItWasCapturedAndItsStatus(t *testing.T) {
	t.Parallel()

	capturedAt := time.Date(2026, 10, 7, 12, 30, 0, 0, time.UTC)
	writtenPage := storedPage{
		address: "https://example.org/refused", contentType: "text/html", body: []byte("page"),
		capturedAt: capturedAt, capturedBy: "4play Firefox/152.0", status: http.StatusNotFound,
	}

	readPages := pagesOfWARC(t, warcOf(t, []storedPage{writtenPage}))

	if len(readPages) != 1 || !readPages[0].capturedAt.Equal(capturedAt) ||
		readPages[0].capturedBy != writtenPage.capturedBy || readPages[0].status != http.StatusNotFound {
		t.Fatalf("the warc holds %+v, want the page captured at %v by %q with the status 404",
			readPages, capturedAt, writtenPage.capturedBy)
	}
}

func TestAWARCOfNoPageHoldsNoPage(t *testing.T) {
	t.Parallel()

	if readPages := pagesOfWARC(t, warcOf(t, nil)); len(readPages) != 0 {
		t.Fatalf("the warc holds %d pages, want none", len(readPages))
	}
}

func TestEachRecordOfAWARCNamesItsTypeAndTheAddressItHolds(t *testing.T) {
	t.Parallel()

	warc := string(warcOf(t, []storedPage{
		{address: "https://example.org/", contentType: "text/html", body: []byte("hello")},
	}))

	for _, line := range []string{
		"WARC/1.1",
		"WARC-Type: response",
		"WARC-Target-URI: https://example.org/",
		"Content-Type: application/http;msgtype=response",
		"HTTP/1.1 200 OK",
	} {
		if !strings.Contains(warc, line) {
			t.Fatalf("the warc reads %q, want it to hold %q", warc, line)
		}
	}
}
