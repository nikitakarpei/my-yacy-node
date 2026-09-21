package judgedqueries_test

import (
	"bytes"
	"strings"
	"testing"
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
