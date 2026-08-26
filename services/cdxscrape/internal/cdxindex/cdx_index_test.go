package cdxindex_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
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

func TestCapturesForReadsEveryCaptureTheIndexLists(t *testing.T) {
	index := indexServing(t, firstRow+"\n"+secondRow+"\n", nil)

	captures, err := index.CapturesFor(context.Background(), cdxindex.Query{URL: "example.com"})
	if err != nil {
		t.Fatalf("captures for query: %v", err)
	}

	wanted := []cdxindex.Capture{
		{URLKey: "com,example)/", Timestamp: "20240101120000", OriginalURL: "https://example.com/"},
		{
			URLKey:      "com,example)/about",
			Timestamp:   "20240102130000",
			OriginalURL: "https://example.com/about",
		},
	}
	if len(captures) != len(wanted) {
		t.Fatalf("captures = %v, want %v", captures, wanted)
	}
	for at, capture := range captures {
		if capture != wanted[at] {
			t.Fatalf("capture %d = %v, want %v", at, capture, wanted[at])
		}
	}
}

func TestCapturesForAsksTheIndexForTheStatedQuery(t *testing.T) {
	var asked url.Values
	index := indexServing(t, "", func(request *http.Request) {
		asked = request.URL.Query()
		if wanted := "/" + collection + "/cdx"; request.URL.Path != wanted {
			t.Errorf("index path = %q, want %q", request.URL.Path, wanted)
		}
	})

	if _, err := index.CapturesFor(context.Background(), cdxindex.Query{
		URL:        "example.com",
		MatchType:  "domain",
		MediaType:  "text/html",
		StatusCode: 200,
		From:       "2024",
		To:         "2025",
		Limit:      50,
	}); err != nil {
		t.Fatalf("captures for query: %v", err)
	}

	for parameter, wanted := range map[string]string{
		"url":       "example.com",
		"output":    "json",
		"matchType": "domain",
		"from":      "2024",
		"to":        "2025",
		"limit":     "50",
	} {
		if asked.Get(parameter) != wanted {
			t.Errorf("%s = %q, want %q", parameter, asked.Get(parameter), wanted)
		}
	}
	if filters := strings.Join(asked["filter"], " "); filters != "mime:text/html =status:200" {
		t.Errorf("filter = %q, want the media type and the status code", filters)
	}
}

func TestCapturesForLeavesUnstatedBoundsToTheArchive(t *testing.T) {
	var asked url.Values
	index := indexServing(t, "", func(request *http.Request) { asked = request.URL.Query() })

	if _, err := index.CapturesFor(
		context.Background(),
		cdxindex.Query{URL: "example.com"},
	); err != nil {
		t.Fatalf("captures for query: %v", err)
	}

	for _, parameter := range []string{"matchType", "filter", "from", "to", "limit"} {
		if _, stated := asked[parameter]; stated {
			t.Errorf("%s = %q, want it left unstated", parameter, asked[parameter])
		}
	}
}

func TestCapturesForReadsNoCaptureFromAnEmptyIndex(t *testing.T) {
	index := indexServing(t, "", nil)

	captures, err := index.CapturesFor(context.Background(), cdxindex.Query{URL: "example.com"})
	if err != nil {
		t.Fatalf("captures for query: %v", err)
	}
	if len(captures) != 0 {
		t.Fatalf("captures = %v, want none", captures)
	}
}

func TestCapturesForFailsWhenTheArchiveRefusesTheQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		},
	))
	t.Cleanup(server.Close)

	_, err := indexAt(t, server.URL).CapturesFor(
		context.Background(),
		cdxindex.Query{URL: "example.com"},
	)
	if err == nil {
		t.Fatal("captures for query: want an error naming the answer")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error = %v, want it to name the answer", err)
	}
}

func TestCapturesForFailsWhenTheIndexStatesAnUnreadableCapture(t *testing.T) {
	index := indexServing(t, firstRow+"\nnot a capture\n", nil)

	if _, err := index.CapturesFor(
		context.Background(),
		cdxindex.Query{URL: "example.com"},
	); err == nil {
		t.Fatal("captures for query: want an error")
	}
}

func TestCapturesForFailsWhenTheArchiveIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	))
	server.Close()

	if _, err := indexAt(t, server.URL).CapturesFor(
		context.Background(),
		cdxindex.Query{URL: "example.com"},
	); err == nil {
		t.Fatal("captures for query: want an error")
	}
}

func indexServing(
	t *testing.T,
	rows string,
	readQuery func(request *http.Request),
) *cdxindex.CDXIndex {
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
	return indexAt(t, server.URL)
}

func indexAt(t *testing.T, archiveURL string) *cdxindex.CDXIndex {
	t.Helper()
	parsed, err := url.Parse(archiveURL)
	if err != nil {
		t.Fatalf("parse archive url %q: %v", archiveURL, err)
	}
	return cdxindex.New(http.DefaultClient, parsed, collection)
}
