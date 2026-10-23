package http_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlprogress"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawls"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	crawlerapishttp "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlerapis/http"
)

var errServerDown = errors.New("server down")

var (
	firstCrawlID = crawlid.CrawlID{1}
	firstCrawl   = "/crawls/" + firstCrawlID.String()
	unknownCrawl = "/crawls/" + crawlid.CrawlID{2}.String()
)

func crawlIDText(n byte) string {
	return crawlid.CrawlID{n}.String()
}

type fakeRecords struct {
	crawls       map[crawlid.CrawlID]crawls.Crawl
	listedOnly   map[crawlid.CrawlID]yacycrawlcontract.CrawlState
	crawlsOf     map[string]crawlid.CrawlID
	started      []acceptedorder.AcceptedOrder
	paused       []crawlid.CrawlID
	resumed      []crawlid.CrawlID
	askedStates  []yacycrawlcontract.CrawlState
	askedLimit   int
	running      int
	startErr     error
	readErr      error
	changeErr    error
	changedState yacycrawlcontract.CrawlState
}

func newRecords() *fakeRecords {
	return &fakeRecords{
		crawls:     map[crawlid.CrawlID]crawls.Crawl{},
		listedOnly: map[crawlid.CrawlID]yacycrawlcontract.CrawlState{},
		crawlsOf:   map[string]crawlid.CrawlID{},
	}
}

func (r *fakeRecords) Start(
	_ *vault.Txn,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, bool, error) {
	if r.startErr != nil {
		return crawlid.CrawlID{}, false, r.startErr
	}
	if crawlID, known := r.crawlsOf[order.OrderID()]; known {
		return crawlID, false, nil
	}
	crawlID := crawlid.CrawlID{1}
	for range r.crawlsOf {
		crawlID[0]++
	}
	r.crawlsOf[order.OrderID()] = crawlID
	r.started = append(r.started, order)
	r.crawls[crawlID] = crawls.Crawl{
		Order: order.CrawlOrder(),
		State: yacycrawlcontract.CrawlStateRunning,
	}
	return crawlID, true, nil
}

func (r *fakeRecords) CrawlOf(
	_ *vault.Txn,
	crawlID crawlid.CrawlID,
) (crawls.Crawl, bool, error) {
	if r.readErr != nil {
		return crawls.Crawl{}, false, r.readErr
	}
	crawl, found := r.crawls[crawlID]
	return crawl, found, nil
}

func (r *fakeRecords) changed(crawlID crawlid.CrawlID) (yacycrawlcontract.CrawlState, bool, error) {
	if r.changeErr != nil {
		return "", false, r.changeErr
	}
	if _, found := r.crawls[crawlID]; !found {
		return "", false, nil
	}
	return r.changedState, true, nil
}

func (r *fakeRecords) Cancel(
	_ *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.CrawlState, bool, error) {
	return r.changed(crawlID)
}

func (r *fakeRecords) Pause(
	_ *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.CrawlState, bool, error) {
	r.paused = append(r.paused, crawlID)
	return r.changed(crawlID)
}

func (r *fakeRecords) Resume(
	_ *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.CrawlState, bool, error) {
	r.resumed = append(r.resumed, crawlID)
	return r.changed(crawlID)
}

func (r *fakeRecords) AmountOfRunningCrawls(*vault.Txn) (int, error) {
	return r.running, r.readErr
}

func (r *fakeRecords) CrawlIDsOlderThan(
	_ *vault.Txn,
	state yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	limit int,
) ([]crawlid.CrawlID, error) {
	inState := r.crawlIDsIn(state, func(crawlID crawlid.CrawlID) bool {
		return border == (crawlid.CrawlID{}) || crawlID.String() < border.String()
	}, limit)
	slices.Reverse(inState)
	return inState[:min(len(inState), limit)], r.readErr
}

func (r *fakeRecords) CrawlIDsNewerThan(
	_ *vault.Txn,
	state yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	limit int,
) ([]crawlid.CrawlID, error) {
	inState := r.crawlIDsIn(state, func(crawlID crawlid.CrawlID) bool {
		return crawlID.String() > border.String()
	}, limit)
	return inState[:min(len(inState), limit)], r.readErr
}

func (r *fakeRecords) crawlIDsIn(
	state yacycrawlcontract.CrawlState,
	beyondBorder func(crawlid.CrawlID) bool,
	limit int,
) []crawlid.CrawlID {
	r.askedStates, r.askedLimit = append(r.askedStates, state), limit
	var inState []crawlid.CrawlID
	for crawlID, crawl := range r.crawls {
		if crawl.State == state && beyondBorder(crawlID) {
			inState = append(inState, crawlID)
		}
	}
	for crawlID, listedState := range r.listedOnly {
		if listedState == state && beyondBorder(crawlID) {
			inState = append(inState, crawlID)
		}
	}
	slices.SortFunc(inState, func(a, b crawlid.CrawlID) int {
		return strings.Compare(a.String(), b.String())
	})
	return inState
}

type fakePending struct {
	pending  []pendingpagevisit.PendingPageVisit
	amountOf map[crawlid.CrawlID]int
	ofHosts  map[string]int
	readErr  error
}

func (o *fakePending) AddSeeds(
	_ *vault.Txn,
	pageVisits []pendingpagevisit.PendingPageVisit,
) error {
	o.pending = append(o.pending, pageVisits...)
	return nil
}

func (o *fakePending) AmountPendingOf(_ *vault.Txn, crawlID crawlid.CrawlID) (int, error) {
	return o.amountOf[crawlID], o.readErr
}

func (o *fakePending) AmountsPendingByHostOf(
	_ *vault.Txn,
	_ crawlid.CrawlID,
	fn func(host string, amount int),
) error {
	for host, amount := range o.ofHosts {
		fn(host, amount)
	}
	return o.readErr
}

type fakeProgress struct {
	ofCrawl map[crawlid.CrawlID]crawlprogress.Progress
	ofHosts map[string]crawlprogress.Progress
	readErr error
}

func (p *fakeProgress) ProgressOf(
	_ *vault.Txn,
	crawlID crawlid.CrawlID,
) (crawlprogress.Progress, error) {
	return p.ofCrawl[crawlID], p.readErr
}

func (p *fakeProgress) ProgressByHostOf(
	*vault.Txn,
	crawlid.CrawlID,
) (map[string]crawlprogress.Progress, error) {
	return p.ofHosts, p.readErr
}

type fakeFrontier struct {
	paused  bool
	pending int64
	err     error
}

func (f *fakeFrontier) AmountPending(context.Context) (int64, bool) {
	return f.pending, true
}

func (f *fakeFrontier) Pause(context.Context) error {
	f.paused = true
	return f.err
}

func (f *fakeFrontier) Resume(context.Context) error {
	f.paused = false
	return f.err
}

func (f *fakeFrontier) IsPaused(context.Context) (bool, error) {
	return f.paused, f.err
}

type recordedFailures struct {
	requests []string
	answers  []string
}

func (r *recordedFailures) CrawlerAPIRequestFailed(_ context.Context, path string, _ error) {
	r.requests = append(r.requests, path)
}

func (r *recordedFailures) CrawlerAPIAnswerFailed(_ context.Context, path string, _ error) {
	r.answers = append(r.answers, path)
}

type apiScene struct {
	storage  *vault.Vault
	records  *fakeRecords
	pending  *fakePending
	progress *fakeProgress
	frontier *fakeFrontier
	failures *recordedFailures
}

func newScene(t *testing.T) *apiScene {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return &apiScene{
		storage: storage,
		records: newRecords(),
		pending: &fakePending{amountOf: map[crawlid.CrawlID]int{}, ofHosts: map[string]int{}},
		progress: &fakeProgress{
			ofCrawl: map[crawlid.CrawlID]crawlprogress.Progress{},
			ofHosts: map[string]crawlprogress.Progress{},
		},
		frontier: &fakeFrontier{},
		failures: &recordedFailures{},
	}
}

func (scene *apiScene) handler() http.Handler {
	return crawlerapishttp.New(
		scene.storage, scene.records, scene.pending, scene.progress, scene.frontier, scene.failures,
	)
}

func (scene *apiScene) serve(
	t *testing.T,
	method string,
	target string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	scene.handler().ServeHTTP(recorder, request)
	return recorder
}

func (scene *apiScene) keep(t *testing.T, n byte, state yacycrawlcontract.CrawlState) {
	t.Helper()
	scene.records.crawls[crawlid.CrawlID{n}] = crawls.Crawl{
		Order: yacycrawlcontract.CrawlOrder{
			OrderID: "order " + crawlIDText(n),
			SeedURLs: []canonicalurl.CanonicalURL{
				canonicalurltest.CanonicalURLOf(t, "http://host/"+crawlIDText(n)),
			},
		},
		State: state,
	}
}

func requireStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d (%s), want %d", recorder.Code, recorder.Body.String(), want)
	}
}

func answerOf[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var answer T
	if err := json.Unmarshal(recorder.Body.Bytes(), &answer); err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	return answer
}

const placedOrder = `{"SeedURLs":["http://host/"],"Profile":{"Scope":"domain","MaxDepth":1}}`

func TestAPlacedOrderStartsItsCrawlOnceWithItsSeedsPending(t *testing.T) {
	scene := newScene(t)

	first := scene.serve(t, http.MethodPut, "/orders/o1", placedOrder)
	again := scene.serve(t, http.MethodPut, "/orders/o1", placedOrder)

	requireStatus(t, first, http.StatusCreated)
	requireStatus(t, again, http.StatusOK)
	if crawl := answerOf[yacycrawlcontract.Crawl](
		t,
		first,
	); crawl.CrawlID != firstCrawlID.String() ||
		crawl.OrderID != "o1" ||
		crawl.State != yacycrawlcontract.CrawlStateRunning {
		t.Fatalf("answer = %+v, want the first crawl, of o1, running", crawl)
	}
	if crawl := answerOf[yacycrawlcontract.Crawl](
		t,
		again,
	); crawl.CrawlID != firstCrawlID.String() {
		t.Fatalf("answer = %+v, want the first crawl again", crawl)
	}
	if len(scene.records.started) != 1 || len(scene.pending.pending) != 1 ||
		scene.pending.pending[0].CrawlID != firstCrawlID {
		t.Fatalf("started %v pending %v, want one crawl of o1 with its seed pending",
			scene.records.started, scene.pending.pending)
	}
}

func TestAnOrderTheCrawlerCannotReadIsRefused(t *testing.T) {
	for name, body := range map[string]string{
		"undecodable":       `{`,
		"other order ID":    `{"OrderID":"o2","SeedURLs":["http://host/"],"Profile":{"Scope":"domain"}}`,
		"without seeds":     `{"Profile":{"Scope":"domain"}}`,
		"unknown scope":     `{"SeedURLs":["http://host/"],"Profile":{"Scope":"galaxy"}}`,
		"negative page cap": `{"SeedURLs":["http://host/"],"Profile":{"Scope":"domain","MaxPagesPerHost":-1}}`,
	} {
		t.Run(name, func(t *testing.T) {
			scene := newScene(t)

			requireStatus(
				t,
				scene.serve(t, http.MethodPut, "/orders/o1", body),
				http.StatusBadRequest,
			)
			if len(scene.records.started) != 0 {
				t.Fatal("a refused order started a crawl")
			}
		})
	}
}

func TestAnOrderWhoseCrawlCannotStartOrBeReadFails(t *testing.T) {
	for name, records := range map[string]*fakeRecords{
		"start": {startErr: errServerDown},
		"read":  {readErr: errServerDown},
	} {
		t.Run(name, func(t *testing.T) {
			scene := newScene(t)
			scene.records.startErr, scene.records.readErr = records.startErr, records.readErr

			requireStatus(t, scene.serve(t, http.MethodPut, "/orders/o1", placedOrder),
				http.StatusInternalServerError)
			if len(scene.failures.requests) != 1 {
				t.Fatalf("failures = %v, want one", scene.failures.requests)
			}
		})
	}
}

func crawlIDsOf(page yacycrawlcontract.CrawlPage) []string {
	crawlIDs := make([]string, 0, len(page.Crawls))
	for _, crawl := range page.Crawls {
		crawlIDs = append(crawlIDs, crawl.CrawlID)
	}
	return crawlIDs
}

func TestTheCrawlListAnswersPagesWithCursorsBothWays(t *testing.T) {
	scene := newScene(t)
	for _, n := range []byte{4, 3, 2, 1} {
		scene.keep(t, n, yacycrawlcontract.CrawlStateRunning)
	}

	newest := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?state=running&state=paused&limit=2", ""))
	if !slices.Equal(scene.records.askedStates, []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateRunning, yacycrawlcontract.CrawlStatePaused,
	}) {
		t.Fatalf("asked for %v, want both states", scene.records.askedStates)
	}
	older := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?limit=2&cursor="+newest.OlderCursor, ""))
	newer := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?limit=2&cursor="+older.NewerCursor, ""))

	for name, page := range map[string]struct {
		page               yacycrawlcontract.CrawlPage
		want               []string
		hasOlder, hasNewer bool
	}{
		"newest": {newest, []string{crawlIDText(4), crawlIDText(3)}, true, false},
		"older":  {older, []string{crawlIDText(2), crawlIDText(1)}, false, true},
		"newer":  {newer, []string{crawlIDText(4), crawlIDText(3)}, true, false},
	} {
		if got := crawlIDsOf(page.page); !slices.Equal(got, page.want) ||
			(page.page.OlderCursor != "") != page.hasOlder ||
			(page.page.NewerCursor != "") != page.hasNewer {
			t.Errorf("%s page = %v older %q newer %q, want %v older %v newer %v",
				name, got, page.page.OlderCursor, page.page.NewerCursor,
				page.want, page.hasOlder, page.hasNewer)
		}
	}
}

func TestANewerPageKeepsTheCrawlsNearestItsCursor(t *testing.T) {
	scene := newScene(t)
	for _, n := range []byte{4, 3, 2, 1} {
		scene.keep(t, n, yacycrawlcontract.CrawlStateRunning)
	}
	newest := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?limit=2", ""))
	older := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?limit=1&cursor="+newest.OlderCursor, ""))

	newer := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?limit=1&cursor="+older.NewerCursor, ""))

	if got := crawlIDsOf(newer); !slices.Equal(got, []string{crawlIDText(3)}) ||
		newer.NewerCursor == "" {
		t.Fatalf("newer page = %v newer %q, want crawl 3 with a newer cursor",
			got, newer.NewerCursor)
	}
}

func TestTheCrawlListMergesTheAskedStatesNewestFirstAndSkipsCrawlsGone(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	scene.keep(t, 2, yacycrawlcontract.CrawlStatePaused)
	scene.keep(t, 3, yacycrawlcontract.CrawlStateSucceeded)
	scene.keep(t, 4, yacycrawlcontract.CrawlStateCancelling)
	scene.records.listedOnly[crawlid.CrawlID{5}] = yacycrawlcontract.CrawlStateSucceeded

	asked := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls?state=running&state=succeeded", ""))
	every := answerOf[yacycrawlcontract.CrawlPage](t,
		scene.serve(t, http.MethodGet, "/crawls", ""))

	if got := crawlIDsOf(asked); !slices.Equal(got, []string{crawlIDText(3), crawlIDText(1)}) {
		t.Fatalf("asked page = %v, want 3 then 1", got)
	}
	if got := crawlIDsOf(every); !slices.Equal(got, []string{
		crawlIDText(4), crawlIDText(3), crawlIDText(2), crawlIDText(1),
	}) || every.Crawls[0].Order != nil {
		t.Fatalf("every page = %v, want 4 to 1 without their orders", got)
	}
}

func TestAPageAtBothEndsOfTheCrawlListCarriesNoCursor(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)

	page := answerOf[yacycrawlcontract.CrawlPage](t, scene.serve(t, http.MethodGet, "/crawls", ""))

	if page.OlderCursor != "" || page.NewerCursor != "" || scene.records.askedLimit != 21 {
		t.Fatalf(
			"page = %+v read with limit %d, want no cursors and the default limit and one more",
			page,
			scene.records.askedLimit,
		)
	}
}

func TestAnEmptyPageOfTheCrawlListCarriesNoCursor(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	beyondTheTop := "/crawls?cursor=" + cursorOf("newer|"+crawlIDText(9))

	page := answerOf[yacycrawlcontract.CrawlPage](
		t,
		scene.serve(t, http.MethodGet, beyondTheTop, ""),
	)

	if len(page.Crawls) != 0 || page.OlderCursor != "" || page.NewerCursor != "" {
		t.Fatalf("page = %+v, want no crawl and no cursor", page)
	}
}

func TestACrawlListQueryTheCrawlerCannotReadIsRefused(t *testing.T) {
	for _, query := range []string{
		"limit=0", "limit=101", "limit=many", "state=lost", "cursor=***",
		"cursor=" + cursorOf("sideways|"+crawlIDText(1)), "cursor=" + cursorOf("older|soon"),
		"cursor=" + cursorOf("older"),
	} {
		t.Run(query, func(t *testing.T) {
			requireStatus(
				t,
				newScene(t).serve(t, http.MethodGet, "/crawls?"+query, ""),
				http.StatusBadRequest,
			)
		})
	}
}

func cursorOf(text string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(text))
}

func TestACrawlListTheCrawlerCannotReadFails(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	listUnread := newScene(t)
	listUnread.records.readErr = errServerDown
	progressUnread := newScene(t)
	progressUnread.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	progressUnread.progress.readErr = errServerDown
	pendingUnread := newScene(t)
	pendingUnread.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	pendingUnread.pending.readErr = errServerDown

	for name, failing := range map[string]*apiScene{
		"list": listUnread, "progress": progressUnread, "pending": pendingUnread,
	} {
		t.Run(name, func(t *testing.T) {
			requireStatus(t, failing.serve(t, http.MethodGet, "/crawls", ""),
				http.StatusInternalServerError)
		})
	}
}

func TestACrawlIsShownWithItsPendingVisitsAndProgress(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateCancelling)
	scene.pending.amountOf[firstCrawlID] = 3
	scene.progress.ofCrawl[firstCrawlID] = crawlprogress.Progress{
		Completed: map[yacycrawlcontract.PageVisitResult]int64{
			yacycrawlcontract.ResultPublished: 2,
		},
		Dropped: map[yacycrawlcontract.PageVisitDropReason]int64{
			yacycrawlcontract.DropReasonCrawlCancelled: 4,
		},
		UnadmittedLinks: 5,
	}

	known := scene.serve(t, http.MethodGet, firstCrawl, "")
	unknown := scene.serve(t, http.MethodGet, unknownCrawl, "")
	unreadable := scene.serve(t, http.MethodGet, "/crawls/not-a-crawl-id", "")

	requireStatus(t, known, http.StatusOK)
	requireStatus(t, unknown, http.StatusNotFound)
	requireStatus(t, unreadable, http.StatusNotFound)
	crawl := answerOf[yacycrawlcontract.Crawl](t, known)
	if crawl.State != yacycrawlcontract.CrawlStateCancelling || crawl.PendingPageVisits != 3 ||
		crawl.CompletedPageVisitsByResult[yacycrawlcontract.ResultPublished] != 2 ||
		crawl.DroppedPageVisitsByReason[yacycrawlcontract.DropReasonCrawlCancelled] != 4 ||
		crawl.UnadmittedLinks != 5 || crawl.AmountOfSeedURLs != 1 ||
		crawl.FirstSeedURL.String() != "http://host/"+crawlIDText(1) || crawl.Order == nil {
		t.Fatalf("crawl = %+v, want the cancelling crawl with 3 pending and its progress", crawl)
	}
	scene.records.readErr = errServerDown
	requireStatus(
		t,
		scene.serve(t, http.MethodGet, firstCrawl, ""),
		http.StatusInternalServerError,
	)
}

func TestCancellingACrawlAnswersByTheStateItEndsIn(t *testing.T) {
	for state, want := range map[yacycrawlcontract.CrawlState]int{
		yacycrawlcontract.CrawlStateCancelling: http.StatusNoContent,
		yacycrawlcontract.CrawlStateCancelled:  http.StatusNoContent,
		yacycrawlcontract.CrawlStateSucceeded:  http.StatusConflict,
	} {
		t.Run(string(state), func(t *testing.T) {
			scene := newScene(t)
			scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
			scene.records.changedState = state

			requireStatus(t, scene.serve(t, http.MethodPost, firstCrawl+"/cancel", ""), want)
		})
	}
	requireStatus(
		t,
		newScene(t).serve(t, http.MethodPost, firstCrawl+"/cancel", ""),
		http.StatusNotFound,
	)
	failing := newScene(t)
	failing.records.changeErr = errServerDown
	requireStatus(
		t,
		failing.serve(t, http.MethodPost, firstCrawl+"/cancel", ""),
		http.StatusInternalServerError,
	)
}

func TestPausingACrawlPausesOrResumesItUnlessItIsCancellingOrFinished(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	scene.records.changedState = yacycrawlcontract.CrawlStatePaused

	requireStatus(
		t,
		scene.serve(t, http.MethodPut, firstCrawl+"/pause", `{"Paused":true}`),
		http.StatusOK,
	)
	requireStatus(
		t,
		scene.serve(t, http.MethodPut, firstCrawl+"/pause", `{"Paused":false}`),
		http.StatusOK,
	)
	if !slices.Equal(scene.records.paused, []crawlid.CrawlID{firstCrawlID}) ||
		!slices.Equal(scene.records.resumed, []crawlid.CrawlID{firstCrawlID}) {
		t.Fatalf("paused %v resumed %v, want one of each", scene.records.paused,
			scene.records.resumed)
	}
	for _, state := range []yacycrawlcontract.CrawlState{
		yacycrawlcontract.CrawlStateCancelling,
		yacycrawlcontract.CrawlStateCancelled,
		yacycrawlcontract.CrawlStateSucceeded,
	} {
		scene.records.changedState = state
		requireStatus(
			t,
			scene.serve(t, http.MethodPut, firstCrawl+"/pause", `{"Paused":true}`),
			http.StatusConflict,
		)
	}
	requireStatus(
		t,
		scene.serve(t, http.MethodPut, firstCrawl+"/pause", `{`),
		http.StatusBadRequest,
	)
}

func TestTheCrawlerStatusCountsPendingVisitsAndRunningCrawls(t *testing.T) {
	scene := newScene(t)
	scene.frontier.paused, scene.frontier.pending, scene.records.running = true, 7, 2

	recorder := scene.serve(t, http.MethodGet, "/crawler-status", "")

	requireStatus(t, recorder, http.StatusOK)
	if status := answerOf[yacycrawlcontract.CrawlerStatus](t, recorder); status !=
		(yacycrawlcontract.CrawlerStatus{
			Paused: true, PendingPageVisits: 7, AmountOfRunningCrawls: 2,
		}) {
		t.Fatalf("status = %+v, want paused with 7 pending and 2 running", status)
	}
}

func TestACrawlerStatusTheCrawlerCannotReadFails(t *testing.T) {
	pauseUnread := newScene(t)
	pauseUnread.frontier.err = errServerDown
	countUnread := newScene(t)
	countUnread.records.readErr = errServerDown

	requireStatus(
		t,
		pauseUnread.serve(t, http.MethodGet, "/crawler-status", ""),
		http.StatusInternalServerError,
	)
	requireStatus(
		t,
		countUnread.serve(t, http.MethodGet, "/crawler-status", ""),
		http.StatusInternalServerError,
	)
}

func TestTheCrawlerPausesAndResumesEveryVisit(t *testing.T) {
	scene := newScene(t)

	requireStatus(
		t,
		scene.serve(t, http.MethodPut, "/crawler-pause", `{"Paused":true}`),
		http.StatusOK,
	)
	if !scene.frontier.paused {
		t.Fatal("the crawler did not pause")
	}
	requireStatus(
		t,
		scene.serve(t, http.MethodPut, "/crawler-pause", `{"Paused":false}`),
		http.StatusOK,
	)
	if scene.frontier.paused {
		t.Fatal("the crawler did not resume")
	}
	requireStatus(
		t,
		scene.serve(t, http.MethodPut, "/crawler-pause", `nope`),
		http.StatusBadRequest,
	)
	scene.frontier.err = errServerDown
	requireStatus(
		t,
		scene.serve(t, http.MethodPut, "/crawler-pause", `{"Paused":true}`),
		http.StatusInternalServerError,
	)
}

func TestAStateChangeFromAnotherOriginIsRefused(t *testing.T) {
	scene := newScene(t)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/crawler-pause",
		strings.NewReader(`{"Paused":true}`))
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	recorder := httptest.NewRecorder()

	scene.handler().ServeHTTP(recorder, request)

	requireStatus(t, recorder, http.StatusForbidden)
	if scene.frontier.paused {
		t.Fatal("a cross-origin request paused the crawler")
	}
}

type brokenAnswer struct {
	httptest.ResponseRecorder
}

func (*brokenAnswer) Write([]byte) (int, error) {
	return 0, errServerDown
}

func TestAnAnswerThatCannotBeWrittenIsReported(t *testing.T) {
	scene := newScene(t)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/crawler-status", nil)

	scene.handler().ServeHTTP(&brokenAnswer{ResponseRecorder: *httptest.NewRecorder()}, request)

	if len(scene.failures.answers) != 1 {
		t.Fatalf("answer failures = %v, want one", scene.failures.answers)
	}
}

func hostNamesOf(page yacycrawlcontract.CrawlHostPage) []string {
	names := make([]string, 0, len(page.Hosts))
	for _, host := range page.Hosts {
		names = append(names, host.Host)
	}
	return names
}

func TestTheHostsOfACrawlComeWithTheMostURLsFirst(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	scene.pending.ofHosts = map[string]int{"few": 1, "many": 2}
	scene.progress.ofHosts = map[string]crawlprogress.Progress{
		"many": {
			Completed: map[yacycrawlcontract.PageVisitResult]int64{
				yacycrawlcontract.ResultPublished: 1,
			},
			Dropped: map[yacycrawlcontract.PageVisitDropReason]int64{},
		},
		"done": {
			Completed: map[yacycrawlcontract.PageVisitResult]int64{
				yacycrawlcontract.ResultPublished: 1,
			},
			Dropped: map[yacycrawlcontract.PageVisitDropReason]int64{},
		},
	}

	page := answerOf[yacycrawlcontract.CrawlHostPage](t,
		scene.serve(t, http.MethodGet, firstCrawl+"/hosts", ""))

	if got := hostNamesOf(page); !slices.Equal(got, []string{"many", "done", "few"}) {
		t.Fatalf("hosts = %v, want many, then done and few by name", got)
	}
	if many := page.Hosts[0]; many.PendingPageVisits != 2 ||
		many.CompletedPageVisitsByResult[yacycrawlcontract.ResultPublished] != 1 {
		t.Fatalf("many = %+v, want 2 pending and 1 published", many)
	}
}

func TestListHostsPagesThroughTheHostsOfACrawl(t *testing.T) {
	scene := newScene(t)
	scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
	scene.pending.ofHosts = map[string]int{"a.example": 1, "b.example": 1, "c.example": 1}

	first := answerOf[yacycrawlcontract.CrawlHostPage](t,
		scene.serve(t, http.MethodGet, firstCrawl+"/hosts?limit=2", ""))
	second := answerOf[yacycrawlcontract.CrawlHostPage](t,
		scene.serve(t, http.MethodGet, firstCrawl+"/hosts?limit=2&cursor="+first.NextCursor, ""))
	back := answerOf[yacycrawlcontract.CrawlHostPage](t, scene.serve(
		t, http.MethodGet, firstCrawl+"/hosts?limit=2&cursor="+second.PreviousCursor, "",
	))
	beyond := answerOf[yacycrawlcontract.CrawlHostPage](t, scene.serve(
		t, http.MethodGet, firstCrawl+"/hosts?limit=2&cursor="+cursorOf("9"), "",
	))

	if got := hostNamesOf(first); !slices.Equal(got, []string{"a.example", "b.example"}) ||
		first.NextCursor == "" || first.PreviousCursor != "" {
		t.Fatalf("first page = %+v", first)
	}
	if got := hostNamesOf(second); !slices.Equal(got, []string{"c.example"}) ||
		second.NextCursor != "" || second.PreviousCursor == "" {
		t.Fatalf("second page = %+v", second)
	}
	if got := hostNamesOf(back); !slices.Equal(got, []string{"a.example", "b.example"}) {
		t.Fatalf("page before the second = %+v", back)
	}
	if len(beyond.Hosts) != 0 {
		t.Fatalf("page beyond the hosts = %+v, want none", beyond)
	}
}

func TestListHostsAnswersNotFoundForAnUnknownCrawl(t *testing.T) {
	scene := newScene(t)

	requireStatus(t, scene.serve(t, http.MethodGet, firstCrawl+"/hosts", ""), http.StatusNotFound)
	requireStatus(t, scene.serve(t, http.MethodGet, "/crawls/not-a-crawl-id/hosts", ""),
		http.StatusNotFound)
}

func TestListHostsRefusesAnUnreadableQuery(t *testing.T) {
	for _, query := range []string{"limit=0", "cursor=%25%25", "cursor=" + "LTE", "cursor=" + "eA"} {
		t.Run(query, func(t *testing.T) {
			requireStatus(t, newScene(t).serve(t, http.MethodGet, firstCrawl+"/hosts?"+query, ""),
				http.StatusBadRequest)
		})
	}
}

func TestListHostsReportsAFailedRead(t *testing.T) {
	for name, failing := range map[string]func(scene *apiScene){
		"record":   func(scene *apiScene) { scene.records.readErr = errServerDown },
		"progress": func(scene *apiScene) { scene.progress.readErr = errServerDown },
		"pending":  func(scene *apiScene) { scene.pending.readErr = errServerDown },
	} {
		t.Run(name, func(t *testing.T) {
			scene := newScene(t)
			scene.keep(t, 1, yacycrawlcontract.CrawlStateRunning)
			failing(scene)

			requireStatus(t, scene.serve(t, http.MethodGet, firstCrawl+"/hosts", ""),
				http.StatusInternalServerError)
			if len(scene.failures.requests) != 1 {
				t.Fatalf("failed requests = %v, want one", scene.failures.requests)
			}
		})
	}
}
