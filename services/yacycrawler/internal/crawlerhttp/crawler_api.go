// Package crawlerhttp serves the crawler's operator API over HTTP: place an order, page
// through the crawls with their progress and through the hosts of a crawl, cancel or pause
// one, and pause or resume every page visit. A state change sent from another origin is
// refused.
package crawlerhttp

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlprogress"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawls"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	requestBodyLimit = 1 << 20
)

var errOrderIDDiffers = errors.New("the body's OrderID differs from the path's")

type CrawlRecords interface {
	Start(
		tx *vault.Txn,
		order acceptedorder.AcceptedOrder,
	) (crawlID crawlid.CrawlID, started bool, err error)
	CrawlOf(tx *vault.Txn, crawlID crawlid.CrawlID) (crawls.Crawl, bool, error)
	Cancel(tx *vault.Txn, crawlID crawlid.CrawlID) (yacycrawlcontract.CrawlState, bool, error)
	Pause(tx *vault.Txn, crawlID crawlid.CrawlID) (yacycrawlcontract.CrawlState, bool, error)
	Resume(tx *vault.Txn, crawlID crawlid.CrawlID) (yacycrawlcontract.CrawlState, bool, error)
	AmountOfRunningCrawls(tx *vault.Txn) (int, error)
	CrawlIDsOlderThan(
		tx *vault.Txn,
		state yacycrawlcontract.CrawlState,
		border crawlid.CrawlID,
		limit int,
	) ([]crawlid.CrawlID, error)
	CrawlIDsNewerThan(
		tx *vault.Txn,
		state yacycrawlcontract.CrawlState,
		border crawlid.CrawlID,
		limit int,
	) ([]crawlid.CrawlID, error)
}

type PendingPageVisits interface {
	AddSeeds(tx *vault.Txn, seedPageVisits []pendingpagevisit.PendingPageVisit) error
	AmountPendingOf(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error)
	AmountsPendingByHostOf(
		tx *vault.Txn,
		crawlID crawlid.CrawlID,
		fn func(host string, amount int),
	) error
}

type ProgressCounts interface {
	ProgressOf(tx *vault.Txn, crawlID crawlid.CrawlID) (crawlprogress.Progress, error)
	ProgressByHostOf(
		tx *vault.Txn,
		crawlID crawlid.CrawlID,
	) (map[string]crawlprogress.Progress, error)
}

type Frontier interface {
	AmountPending(ctx context.Context) (int64, bool)
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	IsPaused(ctx context.Context) (bool, error)
}

type CrawlerAPIObserver interface {
	CrawlerAPIRequestFailed(ctx context.Context, path string, err error)
	CrawlerAPIAnswerFailed(ctx context.Context, path string, err error)
}

type crawlerAPI struct {
	storage  *vault.Vault
	stores   crawlStores
	frontier Frontier
	observer CrawlerAPIObserver
}

//nolint:revive // the API names the vault, the three crawl state stores, the frontier and an observer
func New(
	storage *vault.Vault,
	records CrawlRecords,
	pending PendingPageVisits,
	progress ProgressCounts,
	frontier Frontier,
	observer CrawlerAPIObserver,
) http.Handler {
	api := crawlerAPI{
		storage:  storage,
		stores:   crawlStores{records: records, pending: pending, progress: progress},
		frontier: frontier,
		observer: observer,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(routeOf(http.MethodPut, yacycrawlcontract.OrderPath), api.placeOrder)
	mux.HandleFunc(routeOf(http.MethodGet, yacycrawlcontract.CrawlsPath), api.listCrawls)
	mux.HandleFunc(routeOf(http.MethodGet, yacycrawlcontract.CrawlPath), api.showCrawl)
	mux.HandleFunc(routeOf(http.MethodGet, yacycrawlcontract.CrawlHostsPath), api.listHosts)
	mux.HandleFunc(routeOf(http.MethodPost, yacycrawlcontract.CrawlCancelPath), api.cancelCrawl)
	mux.HandleFunc(routeOf(http.MethodPut, yacycrawlcontract.CrawlPausePath), api.pauseCrawl)
	mux.HandleFunc(
		routeOf(http.MethodGet, yacycrawlcontract.CrawlerStatusPath),
		api.showCrawlerStatus,
	)
	mux.HandleFunc(routeOf(http.MethodPut, yacycrawlcontract.CrawlerPausePath), api.pauseCrawler)
	return http.NewCrossOriginProtection().Handler(mux)
}

func routeOf(method string, path yacycrawlcontract.APIPath) string {
	return method + " " + string(path)
}

func (api crawlerAPI) placeOrder(w http.ResponseWriter, r *http.Request) {
	var sentOrder yacycrawlcontract.CrawlOrder
	if !api.readJSON(w, r, &sentOrder) {
		return
	}
	order, err := orderPlacedAt(r.PathValue(yacycrawlcontract.IDWildcard), sentOrder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	crawlID, started, err := api.startCrawl(r.Context(), order)
	if err != nil {
		api.failed(w, r, err)
		return
	}
	crawl, _, err := api.crawlOf(r.Context(), crawlID)
	if err != nil {
		api.failed(w, r, err)
		return
	}
	api.writeJSON(w, r, statusOfPlacement(started), crawl)
}

func (api crawlerAPI) startCrawl(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, bool, error) {
	var crawlID crawlid.CrawlID
	var started bool
	if err := api.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		crawlID, started, err = api.stores.records.Start(tx, order)
		if err != nil || !started {
			return err
		}
		return api.stores.pending.AddSeeds(
			tx,
			pendingpagevisit.SeedPageVisitsOf(crawlID, order.SeedURLs()),
		)
	}); err != nil {
		return crawlid.CrawlID{}, false, fmt.Errorf(
			"start the crawl of order %s: %w",
			order.OrderID(),
			err,
		)
	}
	return crawlID, started, nil
}

func (api crawlerAPI) crawlOf(
	ctx context.Context,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.Crawl, bool, error) {
	var crawl yacycrawlcontract.Crawl
	var found bool
	if err := api.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		crawl, found, err = api.stores.crawlViewIn(tx, crawlID)
		return err
	}); err != nil {
		return yacycrawlcontract.Crawl{}, false, fmt.Errorf("read the crawl %s: %w", crawlID, err)
	}
	return crawl, found, nil
}

func orderPlacedAt(
	orderID string,
	sentOrder yacycrawlcontract.CrawlOrder,
) (acceptedorder.AcceptedOrder, error) {
	if sentOrder.OrderID != "" && sentOrder.OrderID != orderID {
		return acceptedorder.AcceptedOrder{}, errOrderIDDiffers
	}
	sentOrder.OrderID = orderID
	return acceptedorder.AcceptedOrderFrom(sentOrder)
}

func statusOfPlacement(started bool) int {
	if started {
		return http.StatusCreated
	}
	return http.StatusOK
}

func (api crawlerAPI) listCrawls(w http.ResponseWriter, r *http.Request) {
	query, err := crawlQueryFrom(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var page yacycrawlcontract.CrawlPage
	if err := api.storage.View(r.Context(), func(tx *vault.Txn) error {
		var err error
		page, err = query.cursor.pageIn(tx, api.stores, query.states, query.limit)
		return err
	}); err != nil {
		api.failed(w, r, fmt.Errorf("read the crawl list: %w", err))
		return
	}
	api.writeJSON(w, r, http.StatusOK, page)
}

type crawlQuery struct {
	states []yacycrawlcontract.CrawlState
	cursor crawlCursor
	limit  int
}

var (
	errLimitOutOfRange = errors.New(
		"limit must be a whole number from 1 to " + strconv.Itoa(maxPageLimit),
	)
	errUnknownState = errors.New("unknown crawl state")
)

func crawlQueryFrom(r *http.Request) (crawlQuery, error) {
	values := r.URL.Query()
	limit, err := limitIn(values)
	if err != nil {
		return crawlQuery{}, err
	}
	states := make([]yacycrawlcontract.CrawlState, 0, len(values["state"]))
	for _, value := range values["state"] {
		state := yacycrawlcontract.CrawlState(value)
		if !slices.Contains(yacycrawlcontract.CrawlStates, state) {
			return crawlQuery{}, fmt.Errorf("%w %q", errUnknownState, value)
		}
		states = append(states, state)
	}
	cursor, err := crawlCursorOf(values.Get("cursor"))
	if err != nil {
		return crawlQuery{}, err
	}
	return crawlQuery{states: states, cursor: cursor, limit: limit}, nil
}

func limitIn(values url.Values) (int, error) {
	limit, err := strconv.Atoi(cmp.Or(values.Get("limit"), strconv.Itoa(defaultPageLimit)))
	if err != nil || limit < 1 || limit > maxPageLimit {
		return 0, errLimitOutOfRange
	}
	return limit, nil
}

func (api crawlerAPI) showCrawl(w http.ResponseWriter, r *http.Request) {
	crawlID, known := crawlIDIn(r)
	if !known {
		http.NotFound(w, r)
		return
	}
	crawl, found, err := api.crawlOf(r.Context(), crawlID)
	if err != nil {
		api.failed(w, r, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	api.writeJSON(w, r, http.StatusOK, crawl)
}

func (api crawlerAPI) listHosts(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	limit, err := limitIn(values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	position, err := hostPositionOf(values.Get("cursor"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	crawlID, known := crawlIDIn(r)
	if !known {
		http.NotFound(w, r)
		return
	}
	hosts, found, err := api.hostsOf(r.Context(), crawlID, position, limit+1)
	if err != nil {
		api.failed(w, r, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	api.writeJSON(w, r, http.StatusOK, hostPageFrom(hosts, position, limit))
}

func (api crawlerAPI) hostsOf(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	position int,
	amount int,
) ([]yacycrawlcontract.CrawlHost, bool, error) {
	var ranked []rankedHost
	var found bool
	if err := api.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		if _, found, err = api.stores.records.CrawlOf(tx, crawlID); err != nil || !found {
			return err
		}
		ranked, err = api.stores.rankedHostsIn(tx, crawlID)
		return err
	}); err != nil {
		return nil, false, fmt.Errorf("read the hosts of the crawl %s: %w", crawlID, err)
	}
	shown := make([]yacycrawlcontract.CrawlHost, 0, amount)
	for _, host := range ranked[min(position, len(ranked)):min(position+amount, len(ranked))] {
		shown = append(shown, host.crawlHost)
	}
	return shown, found, nil
}

func hostPageFrom(
	hosts []yacycrawlcontract.CrawlHost,
	position int,
	limit int,
) yacycrawlcontract.CrawlHostPage {
	hostPage := yacycrawlcontract.CrawlHostPage{Hosts: hosts[:min(len(hosts), limit)]}
	if len(hosts) > limit {
		hostPage.NextCursor = hostCursorAt(position + limit)
	}
	if position > 0 {
		hostPage.PreviousCursor = hostCursorAt(max(position-limit, 0))
	}
	return hostPage
}

func (api crawlerAPI) cancelCrawl(w http.ResponseWriter, r *http.Request) {
	crawlID, known := crawlIDIn(r)
	if !known {
		http.NotFound(w, r)
		return
	}
	state, found, err := api.changeState(r.Context(), crawlID, api.stores.records.Cancel)
	if err != nil {
		api.failed(w, r, err)
		return
	}
	if refusedChange(w, r, state, found, yacycrawlcontract.CrawlCancellation) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api crawlerAPI) pauseCrawl(w http.ResponseWriter, r *http.Request) {
	var pause yacycrawlcontract.Pause
	if !api.readJSON(w, r, &pause) {
		return
	}
	crawlID, known := crawlIDIn(r)
	if !known {
		http.NotFound(w, r)
		return
	}
	applyChange, crawlChange := api.stores.records.Resume, yacycrawlcontract.CrawlResumption
	if pause.Paused {
		applyChange, crawlChange = api.stores.records.Pause, yacycrawlcontract.CrawlPause
	}
	state, found, err := api.changeState(r.Context(), crawlID, applyChange)
	if err != nil {
		api.failed(w, r, err)
		return
	}
	if refusedChange(w, r, state, found, crawlChange) {
		return
	}
	api.writeJSON(w, r, http.StatusOK, pause)
}

func (api crawlerAPI) changeState(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	change func(
		tx *vault.Txn,
		crawlID crawlid.CrawlID,
	) (yacycrawlcontract.CrawlState, bool, error),
) (yacycrawlcontract.CrawlState, bool, error) {
	var state yacycrawlcontract.CrawlState
	var found bool
	if err := api.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		state, found, err = change(tx, crawlID)
		return err
	}); err != nil {
		return "", false, fmt.Errorf("change the state of the crawl %s: %w", crawlID, err)
	}
	return state, found, nil
}

func crawlIDIn(r *http.Request) (crawlid.CrawlID, bool) {
	var crawlID crawlid.CrawlID
	return crawlID, crawlID.UnmarshalText([]byte(r.PathValue(yacycrawlcontract.IDWildcard))) == nil
}

func refusedChange(
	w http.ResponseWriter,
	r *http.Request,
	state yacycrawlcontract.CrawlState,
	found bool,
	crawlChange yacycrawlcontract.CrawlStateChange,
) bool {
	switch {
	case !found:
		http.NotFound(w, r)
	case crawlChange.Refuses(state):
		http.Error(w, "the crawl has "+string(state), http.StatusConflict)
	default:
		return false
	}
	return true
}

func (api crawlerAPI) showCrawlerStatus(w http.ResponseWriter, r *http.Request) {
	paused, err := api.frontier.IsPaused(r.Context())
	if err != nil {
		api.failed(w, r, err)
		return
	}
	var running int
	if err := api.storage.View(r.Context(), func(tx *vault.Txn) error {
		var err error
		running, err = api.stores.records.AmountOfRunningCrawls(tx)
		return err
	}); err != nil {
		api.failed(w, r, fmt.Errorf("count the running crawls: %w", err))
		return
	}
	pending, _ := api.frontier.AmountPending(r.Context())
	api.writeJSON(w, r, http.StatusOK, yacycrawlcontract.CrawlerStatus{
		Paused:                paused,
		PendingPageVisits:     pending,
		AmountOfRunningCrawls: int64(running),
	})
}

func (api crawlerAPI) pauseCrawler(w http.ResponseWriter, r *http.Request) {
	var pause yacycrawlcontract.Pause
	if !api.readJSON(w, r, &pause) {
		return
	}
	change := api.frontier.Resume
	if pause.Paused {
		change = api.frontier.Pause
	}
	if err := change(r.Context()); err != nil {
		api.failed(w, r, err)
		return
	}
	api.writeJSON(w, r, http.StatusOK, pause)
}

func (api crawlerAPI) readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, requestBodyLimit))
	if err := decoder.Decode(value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func (api crawlerAPI) writeJSON(w http.ResponseWriter, r *http.Request, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		api.observer.CrawlerAPIAnswerFailed(r.Context(), r.URL.Path, err)
	}
}

func (api crawlerAPI) failed(w http.ResponseWriter, r *http.Request, err error) {
	api.observer.CrawlerAPIRequestFailed(r.Context(), r.URL.Path, err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
