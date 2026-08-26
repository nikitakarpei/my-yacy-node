package pywb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

const (
	collection = "archive"

	firstRow = `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240101120000", ` +
		`"url": "https://example.com/", ` +
		`"mime": "text/html", "status": "200"}`
	secondRow = `{"urlkey": "com,example)/about", ` +
		`"timestamp": "20240102130000", ` +
		`"url": "https://example.com/about", ` +
		`"mime": "text/html", "status": "200"}`
)

func TestNewestCapturesForKeepsTheNewestCaptureOfEveryPage(t *testing.T) {
	newerFirstRow := `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240501120000", ` +
		`"url": "https://example.com/", ` +
		`"mime": "text/html", "status": "200"}`
	archive := archiveServing(
		t,
		strings.Join([]string{firstRow, newerFirstRow, secondRow}, "\n"),
		nil,
	)

	newestCaptures, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	)
	if err != nil {
		t.Fatalf("newest captures for query: %v", err)
	}

	wanted := []pywb.Capture{
		{URLKey: "com,example)/", Timestamp: "20240501120000", OriginalURL: "https://example.com/"},
		{
			URLKey:      "com,example)/about",
			Timestamp:   "20240102130000",
			OriginalURL: "https://example.com/about",
		},
	}
	if len(newestCaptures.Captures) != len(wanted) {
		t.Fatalf("captures = %v, want %v", newestCaptures.Captures, wanted)
	}
	for at, capture := range newestCaptures.Captures {
		if capture != wanted[at] {
			t.Fatalf("capture %d = %v, want %v", at, capture, wanted[at])
		}
	}
	if newestCaptures.CapturesRead != 3 || newestCaptures.HasMorePages {
		t.Fatalf("newest captures = %+v, want all three rows and no more pages", newestCaptures)
	}
}

func TestNewestCapturesForLimitsDistinctPages(t *testing.T) {
	newerFirstRow := `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240501120000", "url": "https://example.com/"}`
	thirdRow := `{"urlkey": "com,example)/contact", ` +
		`"timestamp": "20240103140000", "url": "https://example.com/contact"}`
	archive := archiveServing(
		t,
		strings.Join([]string{firstRow, newerFirstRow, secondRow, thirdRow}, "\n"),
		nil,
	)

	newestCaptures, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		2,
	)
	if err != nil {
		t.Fatalf("newest captures for query: %v", err)
	}

	if len(newestCaptures.Captures) != 2 {
		t.Fatalf("captures = %v, want two distinct pages", newestCaptures.Captures)
	}
	if newestCaptures.Captures[0].Timestamp != "20240501120000" {
		t.Fatalf("first capture = %v, want the newest home page", newestCaptures.Captures[0])
	}
	if newestCaptures.Captures[1].URLKey != "com,example)/about" {
		t.Fatalf("second capture = %v, want the about page", newestCaptures.Captures[1])
	}
	if newestCaptures.CapturesRead != 4 || !newestCaptures.HasMorePages {
		t.Fatalf("newest captures = %+v, want four rows read and more pages", newestCaptures)
	}
}

func TestNewestCapturesForStopsReadingAfterThePageLimit(t *testing.T) {
	archive := archiveServing(t, firstRow+"\n"+secondRow+"\nnot a capture\n", nil)

	newestCaptures, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		1,
	)
	if err != nil {
		t.Fatalf("newest captures for query: %v", err)
	}
	if len(newestCaptures.Captures) != 1 || newestCaptures.CapturesRead != 2 {
		t.Fatalf("newest captures = %+v, want one page from two rows", newestCaptures)
	}
	if !newestCaptures.HasMorePages {
		t.Fatal("newest captures report no more pages")
	}
}

func TestNewestCapturesForAsksTheIndexForTheStatedQuery(t *testing.T) {
	var asked url.Values
	archive := archiveServing(t, "", func(request *http.Request) {
		asked = request.URL.Query()
		if wanted := "/" + collection + "/cdx"; request.URL.Path != wanted {
			t.Errorf("index path = %q, want %q", request.URL.Path, wanted)
		}
	})

	if _, err := archive.NewestCapturesFor(context.Background(), pywb.Query{
		URL:        "example.com",
		MatchType:  "domain",
		MediaType:  "text/html",
		StatusCode: 200,
		From:       "2024",
		To:         "2025",
	}, 50); err != nil {
		t.Fatalf("newest captures for query: %v", err)
	}

	for parameter, wanted := range map[string]string{
		"url":       "example.com",
		"output":    "json",
		"matchType": "domain",
		"from":      "2024",
		"to":        "2025",
	} {
		if asked.Get(parameter) != wanted {
			t.Errorf("%s = %q, want %q", parameter, asked.Get(parameter), wanted)
		}
	}
	if filters := strings.Join(asked["filter"], " "); filters != "mime:text/html =status:200" {
		t.Errorf("filter = %q, want the media type and the status code", filters)
	}
	if _, stated := asked["limit"]; stated {
		t.Errorf("limit = %q, want page limit kept out of the cdx query", asked["limit"])
	}
}

func TestNewestCapturesForLeavesUnstatedBoundsToTheArchive(t *testing.T) {
	var asked url.Values
	archive := archiveServing(t, "", func(request *http.Request) { asked = request.URL.Query() })

	if _, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	); err != nil {
		t.Fatalf("newest captures for query: %v", err)
	}

	for _, parameter := range []string{"matchType", "filter", "from", "to", "limit"} {
		if _, stated := asked[parameter]; stated {
			t.Errorf("%s = %q, want it left unstated", parameter, asked[parameter])
		}
	}
}

func TestNewestCapturesForReadsNoCaptureFromAnEmptyIndex(t *testing.T) {
	archive := archiveServing(t, "", nil)

	newestCaptures, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	)
	if err != nil {
		t.Fatalf("newest captures for query: %v", err)
	}
	if len(newestCaptures.Captures) != 0 || newestCaptures.CapturesRead != 0 {
		t.Fatalf("newest captures = %+v, want none", newestCaptures)
	}
}

func TestNewestCapturesForFailsWhenTheArchiveRefusesTheQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		},
	))
	t.Cleanup(server.Close)

	_, err := archiveAt(t, server.URL).NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	)
	if err == nil {
		t.Fatal("captures for query: want an error naming the answer")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error = %v, want it to name the answer", err)
	}
}

func TestNewestCapturesForFailsWhenTheIndexStatesAnUnreadableCapture(t *testing.T) {
	archive := archiveServing(t, firstRow+"\nnot a capture\n", nil)

	if _, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	); err == nil {
		t.Fatal("newest captures for query: want an error")
	}
}

func TestNewestCapturesForFailsWhenURLKeysAreOutOfOrder(t *testing.T) {
	archive := archiveServing(t, secondRow+"\n"+firstRow+"\n", nil)

	if _, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	); err == nil {
		t.Fatal("newest captures for query: want an error")
	}
}

func TestNewestCapturesForRefusesANegativePageLimit(t *testing.T) {
	archive := archiveServing(t, firstRow+"\n", nil)

	if _, err := archive.NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		-1,
	); err == nil {
		t.Fatal("newest captures for query: want an error")
	}
}

func TestNewestCapturesForFailsWhenTheArchiveIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	))
	server.Close()

	if _, err := archiveAt(t, server.URL).NewestCapturesFor(
		context.Background(),
		pywb.Query{URL: "example.com"},
		0,
	); err == nil {
		t.Fatal("newest captures for query: want an error")
	}
}

func archiveServing(
	t *testing.T,
	rows string,
	readQuery func(request *http.Request),
) *pywb.Archive {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if readQuery != nil {
				readQuery(request)
			}
			_, _ = writer.Write([]byte(rows))
		},
	))
	t.Cleanup(server.Close)
	return archiveAt(t, server.URL)
}

func TestReplayURLOfCarriesTheCollectionTheMomentAndTheCapturedURL(t *testing.T) {
	archive := archiveAt(t, "http://pywb:8080")

	replayURL := replayURLOf(t, archive, pywb.Capture{
		Timestamp:   "20240101120000",
		OriginalURL: "https://example.com/a/page",
	})

	wanted := "http://pywb:8080/archive/20240101120000mp_/" +
		"https://example.com/a/page"
	if replayURL != wanted {
		t.Fatalf("replay url = %q, want %q", replayURL, wanted)
	}
}

func TestReplayURLOfKeepsTheCapturedURLWholeThroughCanonicalForm(t *testing.T) {
	archive := archiveAt(t, "http://pywb:8080")

	for _, capturedURL := range []string{
		"https://example.com/a/page",
		"https://example.com/a/%2E%2E/page",
		"https://example.com/a/page?query=1&other=2",
		"https://example.com/",
	} {
		replayURL := replayURLOf(t, archive, pywb.Capture{
			Timestamp:   "20240101120000",
			OriginalURL: capturedURL,
		})

		readBack := replayURL[len("http://pywb:8080/archive/20240101120000mp_/"):]
		if readBack != capturedURL {
			t.Errorf("replay url carries %q, want %q", readBack, capturedURL)
		}
	}
}

func TestReplayURLOfKeepsATrailingArchivePathElement(t *testing.T) {
	archive := archiveAt(t, "http://pywb:8080/wayback/")

	replayURL := replayURLOf(t, archive, pywb.Capture{
		Timestamp:   "20240101120000",
		OriginalURL: "https://example.com/",
	})

	wanted := "http://pywb:8080/wayback/archive/20240101120000mp_/https://example.com/"
	if replayURL != wanted {
		t.Fatalf("replay url = %q, want %q", replayURL, wanted)
	}
}

func TestReplayURLOfFailsWhenTheArchiveAddressHasNoHost(t *testing.T) {
	archive := archiveAt(t, "http:///")

	if _, err := archive.ReplayURLOf(pywb.Capture{
		Timestamp:   "20240101120000",
		OriginalURL: "https://example.com/",
	}); err == nil {
		t.Fatal("replay url of capture: want an error")
	}
}

func replayURLOf(
	t *testing.T,
	archive *pywb.Archive,
	capture pywb.Capture,
) string {
	t.Helper()
	replayURL, err := archive.ReplayURLOf(capture)
	if err != nil {
		t.Fatalf("replay url of capture %v: %v", capture, err)
	}
	return replayURL.String()
}

func archiveAt(t *testing.T, archiveURL string) *pywb.Archive {
	t.Helper()
	parsed, err := url.Parse(archiveURL)
	if err != nil {
		t.Fatalf("parse archive url %q: %v", archiveURL, err)
	}
	return pywb.New(http.DefaultClient, parsed, collection)
}
