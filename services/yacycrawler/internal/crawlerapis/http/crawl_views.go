package http

import (
	"bytes"
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlprogress"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawls"
)

type crawlStores struct {
	records  CrawlRecords
	pending  PendingPageVisits
	progress ProgressCounts
}

func (stores crawlStores) crawlViewIn(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.Crawl, bool, error) {
	crawl, found, err := stores.records.CrawlOf(tx, crawlID)
	if err != nil || !found {
		return yacycrawlcontract.Crawl{}, false, err
	}
	amountPending, err := stores.pending.AmountPendingOf(tx, crawlID)
	if err != nil {
		return yacycrawlcontract.Crawl{}, false, err
	}
	crawlProgress, err := stores.progress.ProgressOf(tx, crawlID)
	if err != nil {
		return yacycrawlcontract.Crawl{}, false, err
	}
	return crawlViewOf(crawlID, crawl, int64(amountPending), crawlProgress), true, nil
}

func crawlViewOf(
	crawlID crawlid.CrawlID,
	crawl crawls.Crawl,
	pending int64,
	progress crawlprogress.Progress,
) yacycrawlcontract.Crawl {
	order := crawl.Order
	view := yacycrawlcontract.Crawl{
		CrawlID:                     crawlID.String(),
		OrderID:                     order.OrderID,
		Label:                       order.Label,
		StartedAt:                   crawl.StartedAt,
		FinishedAt:                  crawl.FinishedAt,
		State:                       crawl.State,
		AmountOfSeedURLs:            int64(len(order.SeedURLs)),
		PendingPageVisits:           pending,
		UnadmittedLinks:             progress.UnadmittedLinks,
		Order:                       &order,
		CompletedPageVisitsByResult: progress.Completed,
		DroppedPageVisitsByReason:   progress.Dropped,
	}
	if len(order.SeedURLs) > 0 {
		view.FirstSeedURL = order.SeedURLs[0]
	}
	return view
}

func (stores crawlStores) crawlsOlderThan(
	tx *vault.Txn,
	states []yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	amount int,
) ([]yacycrawlcontract.Crawl, error) {
	return stores.crawlsBeside(tx, states, amount, newestFirst, func(
		state yacycrawlcontract.CrawlState,
	) ([]crawlid.CrawlID, error) {
		return stores.records.CrawlIDsOlderThan(tx, state, border, amount)
	})
}

func (stores crawlStores) crawlsNewerThan(
	tx *vault.Txn,
	states []yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	amount int,
) ([]yacycrawlcontract.Crawl, error) {
	newer, err := stores.crawlsBeside(tx, states, amount, oldestFirst, func(
		state yacycrawlcontract.CrawlState,
	) ([]crawlid.CrawlID, error) {
		return stores.records.CrawlIDsNewerThan(tx, state, border, amount)
	})
	slices.Reverse(newer)
	return newer, err
}

func newestFirst(a, b crawlid.CrawlID) int {
	return bytes.Compare(b[:], a[:])
}

func oldestFirst(a, b crawlid.CrawlID) int {
	return bytes.Compare(a[:], b[:])
}

func (stores crawlStores) crawlsBeside(
	tx *vault.Txn,
	states []yacycrawlcontract.CrawlState,
	amount int,
	order func(a, b crawlid.CrawlID) int,
	crawlIDsIn func(state yacycrawlcontract.CrawlState) ([]crawlid.CrawlID, error),
) ([]yacycrawlcontract.Crawl, error) {
	if len(states) == 0 {
		states = yacycrawlcontract.CrawlStates
	}
	var nearest []crawlid.CrawlID
	for _, state := range states {
		crawlIDs, err := crawlIDsIn(state)
		if err != nil {
			return nil, err
		}
		nearest = append(nearest, crawlIDs...)
	}
	slices.SortFunc(nearest, order)
	return stores.crawlsIn(tx, nearest[:min(len(nearest), amount)])
}

func (stores crawlStores) crawlsIn(
	tx *vault.Txn,
	crawlIDs []crawlid.CrawlID,
) ([]yacycrawlcontract.Crawl, error) {
	listed := make([]yacycrawlcontract.Crawl, 0, len(crawlIDs))
	for _, crawlID := range crawlIDs {
		crawl, found, err := stores.crawlViewIn(tx, crawlID)
		if err != nil {
			return nil, err
		}
		if found {
			crawl.Order = nil
			listed = append(listed, crawl)
		}
	}
	return listed, nil
}

type rankedHost struct {
	crawlHost    yacycrawlcontract.CrawlHost
	amountOfURLs int64
}

func (stores crawlStores) rankedHostsIn(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) ([]rankedHost, error) {
	progressByHost, err := stores.progress.ProgressByHostOf(tx, crawlID)
	if err != nil {
		return nil, err
	}
	pendingOfHosts := map[string]int64{}
	if err := stores.pending.AmountsPendingByHostOf(tx, crawlID, func(host string, amount int) {
		pendingOfHosts[host] += int64(amount)
	}); err != nil {
		return nil, err
	}
	for host := range pendingOfHosts {
		if _, counted := progressByHost[host]; !counted {
			progressByHost[host] = crawlprogress.Progress{
				Completed: map[yacycrawlcontract.PageVisitResult]int64{},
				Dropped:   map[yacycrawlcontract.PageVisitDropReason]int64{},
			}
		}
	}
	ranked := make([]rankedHost, 0, len(progressByHost))
	for host, progress := range progressByHost {
		ranked = append(ranked, rankedHostOf(host, pendingOfHosts[host], progress))
	}
	slices.SortFunc(ranked, func(a, b rankedHost) int {
		return cmp.Or(
			cmp.Compare(b.amountOfURLs, a.amountOfURLs),
			cmp.Compare(a.crawlHost.Host, b.crawlHost.Host),
		)
	})
	return ranked, nil
}

func rankedHostOf(host string, pending int64, progress crawlprogress.Progress) rankedHost {
	return rankedHost{
		crawlHost: yacycrawlcontract.CrawlHost{
			Host:                        host,
			PendingPageVisits:           pending,
			CompletedPageVisitsByResult: progress.Completed,
			DroppedPageVisitsByReason:   progress.Dropped,
		},
		amountOfURLs: progress.AmountOfURLs(pending),
	}
}
