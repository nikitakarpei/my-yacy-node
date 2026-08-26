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

func TestNewestReplayURLsForKeepsTheNewestReplayOfEveryPage(t *testing.T) {
	newerFirstRow := `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240501120000", ` +
		`"url": "https://example.com/", ` +
		`"mime": "text/html", "status": "200"}`
	archive, archiveURL := archiveServing(
		t,
		strings.Join([]string{firstRow, newerFirstRow, secondRow}, "\n"),
		nil,
	)

	newestReplayURLs := newestReplayURLsFor(t, archive, 0)

	assertReplayURLs(t, newestReplayURLs, []string{
		archiveURL + "/archive/20240501120000mp_/https://example.com/",
		archiveURL + "/archive/20240102130000mp_/https://example.com/about",
	})
	if newestReplayURLs.CapturesRead != 3 || newestReplayURLs.HasMorePages {
		t.Fatalf(
			"newest replay urls = %+v, want all three rows and no more pages",
			newestReplayURLs,
		)
	}
}

func TestNewestReplayURLsForLimitsDistinctPages(t *testing.T) {
	newerFirstRow := `{"urlkey": "com,example)/", ` +
		`"timestamp": "20240501120000", "url": "https://example.com/"}`
	thirdRow := `{"urlkey": "com,example)/contact", ` +
		`"timestamp": "20240103140000", "url": "https://example.com/contact"}`
	archive, archiveURL := archiveServing(
		t,
		strings.Join([]string{firstRow, newerFirstRow, secondRow, thirdRow}, "\n"),
		nil,
	)

	newestReplayURLs := newestReplayURLsFor(t, archive, 2)

	assertReplayURLs(t, newestReplayURLs, []string{
		archiveURL + "/archive/20240501120000mp_/https://example.com/",
		archiveURL + "/archive/20240102130000mp_/https://example.com/about",
	})
	if newestReplayURLs.CapturesRead != 4 || !newestReplayURLs.HasMorePages {
		t.Fatalf("newest replay urls = %+v, want four rows read and more pages", newestReplayURLs)
	}
}

func TestNewestReplayURLsForStopsReadingAfterThePageLimit(t *testing.T) {
	archive, _ := archiveServing(t, firstRow+"\n"+secondRow+"\nnot a capture\n", nil)

	newestReplayURLs := newestReplayURLsFor(t, archive, 1)

	if len(newestReplayURLs.ReplayURLs) != 1 || newestReplayURLs.CapturesRead != 2 {
		t.Fatalf("newest replay urls = %+v, want one page from two rows", newestReplayURLs)
	}
	if !newestReplayURLs.HasMorePages {
		t.Fatal("newest replay urls report no more pages")
	}
}

func TestNewestReplayURLsForJoinsThePagesOfEveryQuery(t *testing.T) {
	archive, archiveURL := archiveServingPerQueriedURL(t, map[string]string{
		"example.com": firstRow,
		"example.org": otherHostRow,
	})

	newestReplayURLs := newestReplayURLsForBothHosts(t, archive, 0)

	assertReplayURLs(t, newestReplayURLs, []string{
		archiveURL + "/archive/20240101120000mp_/https://example.com/",
		archiveURL + "/archive/20240104150000mp_/https://example.org/",
	})
	if newestReplayURLs.CapturesRead != 2 || newestReplayURLs.HasMorePages {
		t.Fatalf("newest replay urls = %+v, want both hosts and no more pages", newestReplayURLs)
	}
}

func TestNewestReplayURLsForSpendsOnePageLimitAcrossEveryQuery(t *testing.T) {
	archive, archiveURL := archiveServingPerQueriedURL(t, map[string]string{
		"example.com": firstRow + "\n" + secondRow,
		"example.org": otherHostRow + "\n" + otherHostSecondRow,
	})

	newestReplayURLs := newestReplayURLsForBothHosts(t, archive, 3)

	assertReplayURLs(t, newestReplayURLs, []string{
		archiveURL + "/archive/20240101120000mp_/https://example.com/",
		archiveURL + "/archive/20240102130000mp_/https://example.com/about",
		archiveURL + "/archive/20240104150000mp_/https://example.org/",
	})
	if newestReplayURLs.CapturesRead != 4 || !newestReplayURLs.HasMorePages {
		t.Fatalf("newest replay urls = %+v, want four rows read and more pages", newestReplayURLs)
	}
}

func TestNewestReplayURLsForLeavesAQueryUnaskedOnceThePageLimitIsSpent(t *testing.T) {
	askedURLs := []string{}
	archive, _ := archiveServing(t, firstRow, func(request *http.Request) {
		askedURLs = append(askedURLs, request.URL.Query().Get("url"))
	})

	newestReplayURLs := newestReplayURLsForBothHosts(t, archive, 1)

	if !slices.Equal(askedURLs, []string{"example.com"}) {
		t.Errorf("asked urls = %v, want the second query left unasked", askedURLs)
	}
	if len(newestReplayURLs.ReplayURLs) != 1 || !newestReplayURLs.HasMorePages {
		t.Fatalf("newest replay urls = %+v, want one page and more pages", newestReplayURLs)
	}
}

func TestNewestReplayURLsForAsksTheIndexForTheStatedQuery(t *testing.T) {
	var asked url.Values
	archive, _ := archiveServing(t, "", func(request *http.Request) {
		asked = request.URL.Query()
		if wanted := "/" + collection + "/cdx"; request.URL.Path != wanted {
			t.Errorf("index path = %q, want %q", request.URL.Path, wanted)
		}
	})

	if _, err := archive.NewestReplayURLsFor(context.Background(), []pywb.CaptureQuery{{
		URL:        "example.com",
		MatchType:  "domain",
		MediaType:  "text/html",
		StatusCode: 200,
		From:       "2024",
		To:         "2025",
	}}, 50); err != nil {
		t.Fatalf("newest replay urls for query: %v", err)
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

func TestNewestReplayURLsForLeavesUnstatedBoundsToTheArchive(t *testing.T) {
	var asked url.Values
	archive, _ := archiveServing(t, "", func(request *http.Request) {
		asked = request.URL.Query()
	})

	newestReplayURLsFor(t, archive, 0)

	for _, parameter := range []string{"matchType", "filter", "from", "to", "limit"} {
		if _, stated := asked[parameter]; stated {
			t.Errorf("%s = %q, want it left unstated", parameter, asked[parameter])
		}
	}
}

func TestNewestReplayURLsForReadsNoReplayURLFromAnEmptyIndex(t *testing.T) {
	archive, _ := archiveServing(t, "", nil)

	newestReplayURLs := newestReplayURLsFor(t, archive, 0)

	if len(newestReplayURLs.ReplayURLs) != 0 || newestReplayURLs.CapturesRead != 0 {
		t.Fatalf("newest replay urls = %+v, want none", newestReplayURLs)
	}
}

func TestNewestReplayURLsForFailsWhenTheArchiveRefusesTheQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		},
	))
	t.Cleanup(server.Close)

	_, err := archiveAt(t, server.URL).NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	)
	if err == nil {
		t.Fatal("newest replay urls for query: want an error naming the answer")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error = %v, want it to name the answer", err)
	}
}

func TestNewestReplayURLsForFailsWhenTheIndexStatesAnUnreadableCapture(t *testing.T) {
	archive, _ := archiveServing(t, firstRow+"\nnot a capture\n", nil)

	if _, err := archive.NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest replay urls for query: want an error")
	}
}

func TestNewestReplayURLsForFailsWhenURLKeysAreOutOfOrder(t *testing.T) {
	archive, _ := archiveServing(t, secondRow+"\n"+firstRow+"\n", nil)

	if _, err := archive.NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest replay urls for query: want an error")
	}
}

func TestNewestReplayURLsForRefusesANegativePageLimit(t *testing.T) {
	archive, _ := archiveServing(t, firstRow+"\n", nil)

	if _, err := archive.NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		-1,
	); err == nil {
		t.Fatal("newest replay urls for query: want an error")
	}
}

func TestNewestReplayURLsForFailsWhenTheArchiveIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	))
	server.Close()

	if _, err := archiveAt(t, server.URL).NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest replay urls for query: want an error")
	}
}

func TestNewestReplayURLsForKeepsTheCapturedURLWhole(t *testing.T) {
	for _, capturedURL := range []string{
		"https://example.com/a/page",
		"https://example.com/a/%2E%2E/page",
		"https://example.com/a/page?query=1&other=2",
		"https://example.com/",
	} {
		row := `{"urlkey":"com,example)/","timestamp":"20240101120000","url":"` +
			capturedURL + `"}`
		archive, _ := archiveServing(t, row, nil)

		newestReplayURLs := newestReplayURLsFor(t, archive, 0)

		replayURL := newestReplayURLs.ReplayURLs[0].String()
		_, readBack, exists := strings.Cut(replayURL, "mp_/")
		if !exists || readBack != capturedURL {
			t.Errorf("replay url carries %q, want %q", readBack, capturedURL)
		}
	}
}

func TestNewestReplayURLsForKeepsATrailingArchivePathElement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(firstRow))
		},
	))
	t.Cleanup(server.Close)
	archive := archiveAt(t, server.URL+"/wayback/")

	newestReplayURLs := newestReplayURLsFor(t, archive, 0)

	wanted := server.URL +
		"/wayback/archive/20240101120000mp_/https://example.com/"
	assertReplayURLs(t, newestReplayURLs, []string{wanted})
}

func TestNewestReplayURLsForFailsWhenACaptureHasNoReplayURL(t *testing.T) {
	row := `{"urlkey":"com,example)/","timestamp":"20240101120000",` +
		`"url":"https://example.com/%zz"}`
	archive, _ := archiveServing(t, row, nil)

	if _, err := archive.NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		0,
	); err == nil {
		t.Fatal("newest replay urls for query: want an error")
	}
}

func newestReplayURLsFor(
	t *testing.T,
	archive *pywb.Archive,
	pageLimit int,
) pywb.NewestReplayURLs {
	t.Helper()
	newestReplayURLs, err := archive.NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}},
		pageLimit,
	)
	if err != nil {
		t.Fatalf("newest replay urls for query: %v", err)
	}
	return newestReplayURLs
}

func newestReplayURLsForBothHosts(
	t *testing.T,
	archive *pywb.Archive,
	pageLimit int,
) pywb.NewestReplayURLs {
	t.Helper()
	newestReplayURLs, err := archive.NewestReplayURLsFor(
		context.Background(),
		[]pywb.CaptureQuery{{URL: "example.com"}, {URL: "example.org"}},
		pageLimit,
	)
	if err != nil {
		t.Fatalf("newest replay urls for query: %v", err)
	}
	return newestReplayURLs
}

func assertReplayURLs(
	t *testing.T,
	newestReplayURLs pywb.NewestReplayURLs,
	wanted []string,
) {
	t.Helper()
	got := make([]string, 0, len(newestReplayURLs.ReplayURLs))
	for _, replayURL := range newestReplayURLs.ReplayURLs {
		got = append(got, replayURL.String())
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
