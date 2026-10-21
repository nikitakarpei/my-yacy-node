package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	crawlershttp "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/crawlers/http"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const callDeadline = 5 * time.Second

type crawlerRequest struct {
	method string
	path   string
	query  url.Values
	body   string
}

type crawlerAPI struct {
	requests []crawlerRequest
	status   int
	answer   string
}

func crawlerOver(t *testing.T, api *crawlerAPI) *crawlershttp.Crawler {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		api.requests = append(api.requests, crawlerRequest{
			method: r.Method,
			path:   r.URL.EscapedPath(),
			query:  r.URL.Query(),
			body:   string(body),
		})
		w.WriteHeader(api.status)
		_, _ = io.WriteString(w, api.answer)
	}))
	t.Cleanup(server.Close)
	apiURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("parse %q: %v", server.URL, err)
	}
	return crawlershttp.New(apiURL, callDeadline)
}

func unansweringCrawler(t *testing.T) *crawlershttp.Crawler {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	apiURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse %q: %v", server.URL, err)
	}
	server.Close()
	return crawlershttp.New(apiURL, callDeadline)
}

func orderOf(t *testing.T) yacycrawlcontract.CrawlOrder {
	t.Helper()
	seedURL, err := canonicalurl.CanonicalURLOf("https://example.org/")
	if err != nil {
		t.Fatalf("CanonicalURLOf: %v", err)
	}
	return yacycrawlcontract.CrawlOrder{
		OrderID:  "o/1",
		SeedURLs: []canonicalurl.CanonicalURL{seedURL},
		Profile:  yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeDomain},
	}
}

func TestPlacePutsTheOrderUnderItsID(t *testing.T) {
	api := &crawlerAPI{status: http.StatusCreated, answer: `{"CrawlID":"c1"}`}
	order := orderOf(t)

	crawlID, refusal, err := crawlerOver(t, api).Place(t.Context(), order)

	if err != nil || refusal != "" || crawlID != "c1" {
		t.Fatalf("Place = %q, %q, %v; want c1 placed", crawlID, refusal, err)
	}
	request := api.requests[0]
	if request.method != http.MethodPut || request.path != "/orders/o%2F1" {
		t.Fatalf("request = %s %s", request.method, request.path)
	}
	var sent yacycrawlcontract.CrawlOrder
	if err := json.Unmarshal([]byte(request.body), &sent); err != nil {
		t.Fatalf("unmarshal the sent order: %v", err)
	}
	if !reflect.DeepEqual(sent, order) {
		t.Fatalf("sent order = %+v, want %+v", sent, order)
	}
}

func TestPlaceAnswersTheCrawlOfARepeatedOrder(t *testing.T) {
	crawlID, refusal, err := crawlerOver(
		t,
		&crawlerAPI{status: http.StatusOK, answer: `{"CrawlID":"c1"}`},
	).Place(t.Context(), orderOf(t))

	if err != nil || refusal != "" || crawlID != "c1" {
		t.Fatalf("Place = %q, %q, %v; want c1 placed", crawlID, refusal, err)
	}
}

func TestPlaceFailsOnAnAnswerItCannotRead(t *testing.T) {
	_, _, err := crawlerOver(
		t,
		&crawlerAPI{status: http.StatusCreated, answer: "not a crawl"},
	).Place(t.Context(), orderOf(t))

	if err == nil {
		t.Fatal("Place read an unreadable answer")
	}
}

func TestPlaceGivesTheReasonOfARefusal(t *testing.T) {
	cases := []struct {
		name   string
		answer string
		want   string
	}{
		{"with a reason", "unknown crawl scope\n", "unknown crawl scope"},
		{"without a reason", "", "400 Bad Request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := &crawlerAPI{status: http.StatusBadRequest, answer: tc.answer}

			_, refusal, err := crawlerOver(t, api).Place(t.Context(), orderOf(t))

			if err != nil || refusal != tc.want {
				t.Fatalf("Place = %q, %v; want %q", refusal, err, tc.want)
			}
		})
	}
}

func TestPlaceFailsWhenTheCrawlerFails(t *testing.T) {
	_, _, err := crawlerOver(
		t,
		&crawlerAPI{status: http.StatusInternalServerError},
	).Place(t.Context(), orderOf(t))

	if err == nil {
		t.Fatal("Place succeeded on a failed crawler")
	}
}

func TestCrawlsInAsksForTheStatesFromTheCursor(t *testing.T) {
	api := &crawlerAPI{
		status: http.StatusOK,
		answer: `{"Crawls":[{"CrawlID":"c1"}],"OlderCursor":"next"}`,
	}
	states := []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateRunning,
		yacycrawlcontract.CrawlStatePaused,
	}

	page, err := crawlerOver(t, api).CrawlsIn(t.Context(), states, "at", 20)
	if err != nil {
		t.Fatalf("CrawlsIn: %v", err)
	}
	if len(page.Crawls) != 1 || page.Crawls[0].CrawlID != "c1" || page.OlderCursor != "next" {
		t.Fatalf("page = %+v", page)
	}
	query := api.requests[0].query
	if api.requests[0].path != "/crawls" || query.Get("limit") != "20" ||
		query.Get("cursor") != "at" ||
		!slices.Equal(query["state"], []string{"running", "paused"}) {
		t.Fatalf("request = %+v", api.requests[0])
	}
}

func TestCrawlsInLeavesOutAnEmptyCursor(t *testing.T) {
	api := &crawlerAPI{status: http.StatusOK, answer: `{"Crawls":[]}`}

	if _, err := crawlerOver(t, api).CrawlsIn(t.Context(), nil, "", 20); err != nil {
		t.Fatalf("CrawlsIn: %v", err)
	}
	if api.requests[0].query.Has("cursor") {
		t.Fatalf("query = %v, want no cursor", api.requests[0].query)
	}
}

func TestCrawlOfReadsTheCrawl(t *testing.T) {
	api := &crawlerAPI{status: http.StatusOK, answer: `{"CrawlID":"c 1","State":"running"}`}

	crawl, found, err := crawlerOver(t, api).CrawlOf(t.Context(), "c 1")

	if err != nil || !found || crawl.State != yacycrawlcontract.CrawlStateRunning {
		t.Fatalf("CrawlOf = %+v, %t, %v", crawl, found, err)
	}
	if api.requests[0].path != "/crawls/c%201" {
		t.Fatalf("path = %q", api.requests[0].path)
	}
}

func TestCrawlOfFindsNoUnknownCrawl(t *testing.T) {
	_, found, err := crawlerOver(
		t,
		&crawlerAPI{status: http.StatusNotFound},
	).CrawlOf(t.Context(), "c1")

	if err != nil || found {
		t.Fatalf("CrawlOf = %t, %v; want not found", found, err)
	}
}

func TestCrawlOfFailsOnAnAnswerItCannotRead(t *testing.T) {
	_, found, err := crawlerOver(t, &crawlerAPI{status: http.StatusOK, answer: "<html>"}).
		CrawlOf(t.Context(), "c1")

	if err == nil || found {
		t.Fatalf("CrawlOf = %t, %v; want an error", found, err)
	}
}

func TestHostsOfReadsAPageOfHostsFromTheCursor(t *testing.T) {
	api := &crawlerAPI{
		status: http.StatusOK,
		answer: `{"Hosts":[{"Host":"example.org"}],"NextCursor":"n"}`,
	}

	page, err := crawlerOver(t, api).HostsOf(t.Context(), "c 1", "at", 20)

	if err != nil || len(page.Hosts) != 1 || page.Hosts[0].Host != "example.org" ||
		page.NextCursor != "n" {
		t.Fatalf("HostsOf = %+v, %v", page, err)
	}
	request := api.requests[0]
	if request.path != "/crawls/c%201/hosts" || request.query.Get("cursor") != "at" ||
		request.query.Get("limit") != "20" {
		t.Fatalf("request = %+v", request)
	}
}

func TestHostsOfLeavesOutAnEmptyCursor(t *testing.T) {
	api := &crawlerAPI{status: http.StatusOK, answer: `{"Hosts":[]}`}

	if _, err := crawlerOver(t, api).HostsOf(t.Context(), "c1", "", 20); err != nil {
		t.Fatalf("HostsOf: %v", err)
	}
	if api.requests[0].query.Has("cursor") {
		t.Fatalf("query = %v, want no cursor", api.requests[0].query)
	}
}

func TestCrawlerStatusReadsTheStatus(t *testing.T) {
	api := &crawlerAPI{status: http.StatusOK, answer: `{"Paused":true,"PendingPageVisits":4}`}

	status, err := crawlerOver(t, api).CrawlerStatus(t.Context())

	want := yacycrawlcontract.CrawlerStatus{Paused: true, PendingPageVisits: 4}
	if err != nil || status != want {
		t.Fatalf("CrawlerStatus = %+v, %v; want %+v", status, err, want)
	}
}

func TestCrawlerStatusFailsWhenTheCrawlerFails(t *testing.T) {
	if _, err := crawlerOver(
		t,
		&crawlerAPI{status: http.StatusBadGateway},
	).CrawlerStatus(t.Context()); err == nil {
		t.Fatal("CrawlerStatus succeeded on a failed crawler")
	}
}

func TestChangesSendTheirRequests(t *testing.T) {
	api := &crawlerAPI{status: http.StatusOK}
	crawler := crawlerOver(t, api)

	for _, change := range []func() error{
		func() error { return crawler.Cancel(t.Context(), "c1") },
		func() error { return crawler.Pause(t.Context(), "c1") },
		func() error { return crawler.Resume(t.Context(), "c1") },
		func() error { return crawler.PauseCrawler(t.Context()) },
		func() error { return crawler.ResumeCrawler(t.Context()) },
	} {
		if err := change(); err != nil {
			t.Fatalf("change: %v", err)
		}
	}

	want := []crawlerRequest{
		{method: http.MethodPost, path: "/crawls/c1/cancel"},
		{method: http.MethodPut, path: "/crawls/c1/pause", body: `{"Paused":true}` + "\n"},
		{method: http.MethodPut, path: "/crawls/c1/pause", body: `{"Paused":false}` + "\n"},
		{method: http.MethodPut, path: "/crawler-pause", body: `{"Paused":true}` + "\n"},
		{method: http.MethodPut, path: "/crawler-pause", body: `{"Paused":false}` + "\n"},
	}
	for i, request := range api.requests {
		request.query = nil
		if !reflect.DeepEqual(request, want[i]) {
			t.Errorf("request %d = %+v, want %+v", i, request, want[i])
		}
	}
}

func TestChangesThatTheCrawlStateMakesPointlessSucceed(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound, http.StatusConflict} {
		if err := crawlerOver(
			t,
			&crawlerAPI{status: status},
		).Cancel(t.Context(), "c1"); err != nil {
			t.Errorf("Cancel answered %d: %v", status, err)
		}
	}
}

func TestChangesFailWhenTheCrawlerFails(t *testing.T) {
	if err := crawlerOver(
		t,
		&crawlerAPI{status: http.StatusBadRequest},
	).PauseCrawler(t.Context()); err == nil {
		t.Fatal("PauseCrawler succeeded on a refused request")
	}
}

func TestCallsFailWhenTheCrawlerDoesNotAnswer(t *testing.T) {
	crawler := unansweringCrawler(t)

	if _, _, err := crawler.Place(t.Context(), orderOf(t)); err == nil {
		t.Error("Place succeeded")
	}
	if _, err := crawler.CrawlerStatus(t.Context()); err == nil {
		t.Error("CrawlerStatus succeeded")
	}
	if err := crawler.Cancel(t.Context(), "c1"); err == nil {
		t.Error("Cancel succeeded")
	}
}
