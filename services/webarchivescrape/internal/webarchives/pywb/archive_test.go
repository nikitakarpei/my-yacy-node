package pywb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
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
	otherHostRow = `{"urlkey": "org,example)/", ` +
		`"timestamp": "20240104150000", ` +
		`"url": "https://example.org/", ` +
		`"mime": "text/html", "status": "200"}`
	otherHostSecondRow = `{"urlkey": "org,example)/about", ` +
		`"timestamp": "20240105160000", ` +
		`"url": "https://example.org/about", ` +
		`"mime": "text/html", "status": "200"}`
)

func TestNewestArchivedPagesForNamesTheURLEveryCaptureWasTakenFrom(t *testing.T) {
	archive, _ := archiveServing(t, strings.Join([]string{firstRow, secondRow}, "\n"), nil)

	newestArchivedPages := newestArchivedPagesFor(t, archive, 0)

	got := make([]string, 0, len(newestArchivedPages.ArchivedPages))
	for _, archivedPage := range newestArchivedPages.ArchivedPages {
		got = append(got, archivedPage.PageURL.String())
	}
	want := []string{"https://example.com/", "https://example.com/about"}
	if !slices.Equal(got, want) {
		t.Errorf("page urls = %v, want %v", got, want)
	}
}

func TestNewestArchivedPagesForKeepsTheNewestReplayOfEveryPage(t *testing.T) {
	newerFirstRow := `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240501120000", ` +
		`"url": "https://example.com/", ` +
		`"mime": "text/html", "status": "200"}`
	archive, archiveURL := archiveServing(
		t,
		strings.Join([]string{firstRow, newerFirstRow, secondRow}, "\n"),
		nil,
	)

	newestArchivedPages := newestArchivedPagesFor(t, archive, 0)

	assertReplayURLs(t, newestArchivedPages, []string{
		archiveURL + "/archive/20240501120000mp_/https://example.com/",
		archiveURL + "/archive/20240102130000mp_/https://example.com/about",
	})
	if newestArchivedPages.CapturesRead != 3 || newestArchivedPages.HasMorePages {
		t.Fatalf(
			"newest archived pages = %+v, want all three rows and no more pages",
			newestArchivedPages,
		)
	}
}

func TestNewestArchivedPagesForLimitsDistinctPages(t *testing.T) {
	newerFirstRow := `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240501120000", "url": "https://example.com/"}`
	thirdRow := `{"urlkey": "com,example)/contact", ` +
		`"timestamp": "20240103140000", "url": "https://example.com/contact"}`
	archive, archiveURL := archiveServing(
		t,
		strings.Join([]string{firstRow, newerFirstRow, secondRow, thirdRow}, "\n"),
		nil,
	)

	newestArchivedPages := newestArchivedPagesFor(t, archive, 2)

	assertReplayURLs(t, newestArchivedPages, []string{
		archiveURL + "/archive/20240501120000mp_/https://example.com/",
		archiveURL + "/archive/20240102130000mp_/https://example.com/about",
	})
	if newestArchivedPages.CapturesRead != 4 || !newestArchivedPages.HasMorePages {
		t.Fatalf(
			"newest archived pages = %+v, want four rows read and more pages",
			newestArchivedPages,
		)
	}
}

func TestNewestArchivedPagesForStopsReadingAfterThePageLimit(t *testing.T) {
	archive, _ := archiveServing(t, firstRow+"\n"+secondRow+"\nnot a capture\n", nil)

	newestArchivedPages := newestArchivedPagesFor(t, archive, 1)

	if len(newestArchivedPages.ArchivedPages) != 1 || newestArchivedPages.CapturesRead != 2 {
		t.Fatalf("newest archived pages = %+v, want one page from two rows", newestArchivedPages)
	}
	if !newestArchivedPages.HasMorePages {
		t.Fatal("newest replay urls report no more pages")
	}
}

func TestNewestArchivedPagesForJoinsThePagesOfEveryQuery(t *testing.T) {
	archive, archiveURL := archiveServingPerQueriedURL(t, map[string]string{
		"example.com": firstRow,
		"example.org": otherHostRow,
	})

	newestArchivedPages := newestArchivedPagesForBothHosts(t, archive, 0)

	assertReplayURLs(t, newestArchivedPages, []string{
		archiveURL + "/archive/20240101120000mp_/https://example.com/",
		archiveURL + "/archive/20240104150000mp_/https://example.org/",
	})
	if newestArchivedPages.CapturesRead != 2 || newestArchivedPages.HasMorePages {
		t.Fatalf(
			"newest archived pages = %+v, want both hosts and no more pages",
			newestArchivedPages,
		)
	}
}

func TestNewestArchivedPagesForSpendsOnePageLimitAcrossEveryQuery(t *testing.T) {
	archive, archiveURL := archiveServingPerQueriedURL(t, map[string]string{
		"example.com": firstRow + "\n" + secondRow,
		"example.org": otherHostRow + "\n" + otherHostSecondRow,
	})

	newestArchivedPages := newestArchivedPagesForBothHosts(t, archive, 3)

	assertReplayURLs(t, newestArchivedPages, []string{
		archiveURL + "/archive/20240101120000mp_/https://example.com/",
		archiveURL + "/archive/20240102130000mp_/https://example.com/about",
		archiveURL + "/archive/20240104150000mp_/https://example.org/",
	})
	if newestArchivedPages.CapturesRead != 4 || !newestArchivedPages.HasMorePages {
		t.Fatalf(
			"newest archived pages = %+v, want four rows read and more pages",
			newestArchivedPages,
		)
	}
}

func TestNewestArchivedPagesForLeavesAQueryUnaskedOnceThePageLimitIsSpent(t *testing.T) {
	askedURLs := []string{}
	archive, _ := archiveServing(t, firstRow, func(request *http.Request) {
		askedURLs = append(askedURLs, request.URL.Query().Get("url"))
	})

	newestArchivedPages := newestArchivedPagesForBothHosts(t, archive, 1)

	if !slices.Equal(askedURLs, []string{"example.com"}) {
		t.Errorf("asked urls = %v, want the second query left unasked", askedURLs)
	}
	if len(newestArchivedPages.ArchivedPages) != 1 || !newestArchivedPages.HasMorePages {
		t.Fatalf("newest archived pages = %+v, want one page and more pages", newestArchivedPages)
	}
}

func TestNewestArchivedPagesForAsksTheIndexForTheStatedQuery(t *testing.T) {
	var asked url.Values
	archive, _ := archiveServing(t, "", func(request *http.Request) {
		asked = request.URL.Query()
		if wanted := "/" + collection + "/cdx"; request.URL.Path != wanted {
			t.Errorf("index path = %q, want %q", request.URL.Path, wanted)
		}
	})

	if _, err := archive.NewestArchivedPagesFor(context.Background(), []pywb.CaptureQuery{{
		URL:        "example.com",
		MatchType:  "domain",
		MediaType:  "text/html",
		StatusCode: 200,
		From:       "2024",
		To:         "2025",
	}}, 50); err != nil {
		t.Fatalf("newest archived pages for query: %v", err)
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

func TestNewestArchivedPagesForLeavesUnstatedBoundsToTheArchive(t *testing.T) {
	var asked url.Values
	archive, _ := archiveServing(t, "", func(request *http.Request) {
		asked = request.URL.Query()
	})

	newestArchivedPagesFor(t, archive, 0)

	for _, parameter := range []string{"matchType", "filter", "from", "to", "limit"} {
		if _, stated := asked[parameter]; stated {
			t.Errorf("%s = %q, want it left unstated", parameter, asked[parameter])
		}
	}
}

func TestNewestArchivedPagesForReadsNoPageFromAnEmptyIndex(t *testing.T) {
	archive, _ := archiveServing(t, "", nil)

	newestArchivedPages := newestArchivedPagesFor(t, archive, 0)

	if len(newestArchivedPages.ArchivedPages) != 0 || newestArchivedPages.CapturesRead != 0 {
		t.Fatalf("newest archived pages = %+v, want none", newestArchivedPages)
	}
}

func TestNewestArchivedPagesForFailsWhenTheArchiveRefusesTheQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		},
	))
	t.Cleanup(server.Close)

	_, err := archiveAt(t, server.URL).NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	)
	if err == nil {
		t.Fatal("newest archived pages for query: want an error naming the answer")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error = %v, want it to name the answer", err)
	}
}

func TestNewestArchivedPagesForFailsWhenTheIndexStatesAnUnreadableCapture(t *testing.T) {
	archive, _ := archiveServing(t, firstRow+"\nnot a capture\n", nil)

	if _, err := archive.NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest archived pages for query: want an error")
	}
}

func TestNewestArchivedPagesForFailsWhenURLKeysAreOutOfOrder(t *testing.T) {
	archive, _ := archiveServing(t, secondRow+"\n"+firstRow+"\n", nil)

	if _, err := archive.NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest archived pages for query: want an error")
	}
}

func TestNewestArchivedPagesForRefusesANegativePageLimit(t *testing.T) {
	archive, _ := archiveServing(t, firstRow+"\n", nil)

	if _, err := archive.NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		-1,
	); err == nil {
		t.Fatal("newest archived pages for query: want an error")
	}
}

func TestNewestArchivedPagesForFailsWhenTheArchiveIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	))
	server.Close()

	if _, err := archiveAt(t, server.URL).NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest archived pages for query: want an error")
	}
}

func TestNewestArchivedPagesForKeepsTheCapturedURLWhole(t *testing.T) {
	for _, capturedURL := range []string{
		"https://example.com/a/page",
		"https://example.com/a/%2E%2E/page",
		"https://example.com/a/page?query=1&other=2",
		"https://example.com/",
	} {
		row := `{"urlkey":"com,example)/","timestamp":"20240101120000","url":"` +
			capturedURL + `"}`
		archive, _ := archiveServing(t, row, nil)

		newestArchivedPages := newestArchivedPagesFor(t, archive, 0)

		replayURL := newestArchivedPages.ArchivedPages[0].ReplayURL.String()
		_, readBack, exists := strings.Cut(replayURL, "mp_/")
		if !exists || readBack != capturedURL {
			t.Errorf("replay url carries %q, want %q", readBack, capturedURL)
		}
	}
}

func TestNewestArchivedPagesForKeepsATrailingArchivePathElement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(firstRow))
		},
	))
	t.Cleanup(server.Close)
	archive := archiveAt(t, server.URL+"/wayback/")

	newestArchivedPages := newestArchivedPagesFor(t, archive, 0)

	wanted := server.URL +
		"/wayback/archive/20240101120000mp_/https://example.com/"
	assertReplayURLs(t, newestArchivedPages, []string{wanted})
}

func TestNewestArchivedPagesForFailsWhenACaptureHasNoReplayURL(t *testing.T) {
	row := `{"urlkey":"com,example)/","timestamp":"20240101120000",` +
		`"url":"https://example.com/%zz"}`
	archive, _ := archiveServing(t, row, nil)

	if _, err := archive.NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest archived pages for query: want an error")
	}
}

func newestArchivedPagesFor(
	t *testing.T,
	archive *pywb.Archive,
	pageLimit int,
) pywb.NewestArchivedPages {
	t.Helper()
	newestArchivedPages, err := archive.NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		pageLimit,
	)
	if err != nil {
		t.Fatalf("newest archived pages for query: %v", err)
	}
	return newestArchivedPages
}

func newestArchivedPagesForBothHosts(
	t *testing.T,
	archive *pywb.Archive,
	pageLimit int,
) pywb.NewestArchivedPages {
	t.Helper()
	newestArchivedPages, err := archive.NewestArchivedPagesFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}, {URL: "example.org"}},
		pageLimit,
	)
	if err != nil {
		t.Fatalf("newest archived pages for query: %v", err)
	}
	return newestArchivedPages
}

func assertReplayURLs(
	t *testing.T,
	newestArchivedPages pywb.NewestArchivedPages,
	wanted []string,
) {
	t.Helper()
	got := make([]string, 0, len(newestArchivedPages.ArchivedPages))
	for _, archivedPage := range newestArchivedPages.ArchivedPages {
		got = append(got, archivedPage.ReplayURL.String())
	}
	if !slices.Equal(got, wanted) {
		t.Errorf("replay urls = %v, want %v", got, wanted)
	}
}

func archiveServing(
	t *testing.T,
	rows string,
	readQuery func(request *http.Request),
) (*pywb.Archive, string) {
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
	return archiveAt(t, server.URL), server.URL
}

func archiveServingPerQueriedURL(
	t *testing.T,
	rowsByQueriedURL map[string]string,
) (*pywb.Archive, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte(rowsByQueriedURL[request.URL.Query().Get("url")]))
		},
	))
	t.Cleanup(server.Close)
	return archiveAt(t, server.URL), server.URL
}

func archiveAt(t *testing.T, archiveURL string) *pywb.Archive {
	t.Helper()
	parsed, err := url.Parse(archiveURL)
	if err != nil {
		t.Fatalf("parse archive url %q: %v", archiveURL, err)
	}
	return pywb.New(http.DefaultClient, parsed, collection)
}
