// Package crawlprogress counts the progress of each crawl and of each of its hosts: the
// visits it completed by result, the visits it dropped by reason, and the links its profile
// did not admit. A count can miss a visit or count it twice; it is not part of the visit.
package crawlprogress

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type Progress struct {
	Completed       map[yacycrawlcontract.PageVisitResult]int64
	Dropped         map[yacycrawlcontract.PageVisitDropReason]int64
	UnadmittedLinks int64
}

func newProgress() Progress {
	return Progress{
		Completed: map[yacycrawlcontract.PageVisitResult]int64{},
		Dropped:   map[yacycrawlcontract.PageVisitDropReason]int64{},
	}
}

func (progress Progress) AmountOfURLs(pending int64) int64 {
	amount := pending
	for _, completed := range progress.Completed {
		amount += completed
	}
	for _, dropped := range progress.Dropped {
		amount += dropped
	}
	return amount
}

type Counts struct {
	completed       visitCounts[yacycrawlcontract.PageVisitResult]
	dropped         visitCounts[yacycrawlcontract.PageVisitDropReason]
	unadmittedLinks *vault.Amounts[crawlid.CrawlID]
}

func Open(storage *vault.Vault) (*Counts, error) {
	return registerCounts(storage)
}

func (counts *Counts) CountCompleted(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	result yacycrawlcontract.PageVisitResult,
) error {
	if err := counts.completed.raise(tx, crawlID, host, result, 1); err != nil {
		return fmt.Errorf(
			"count a page visit of crawl %s completed as %s: %w",
			crawlID,
			result,
			err,
		)
	}
	return nil
}

func (counts *Counts) CountDropped(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	reason yacycrawlcontract.PageVisitDropReason,
	amount uint,
) error {
	if err := counts.dropped.raise(tx, crawlID, host, reason, amount); err != nil {
		return fmt.Errorf("count page visits of crawl %s dropped as %s: %w", crawlID, reason, err)
	}
	return nil
}

func (counts *Counts) CountUnadmittedLinks(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	amount uint,
) error {
	if err := counts.unadmittedLinks.Raise(tx, crawlID, amount); err != nil {
		return fmt.Errorf("count unadmitted links of crawl %s: %w", crawlID, err)
	}
	return nil
}

func (counts *Counts) ProgressOf(tx *vault.Txn, crawlID crawlid.CrawlID) (Progress, error) {
	completed, err := counts.completed.amountsOf(tx, crawlID)
	if err != nil {
		return Progress{}, fmt.Errorf(
			"read the completed page visits of crawl %s: %w",
			crawlID,
			err,
		)
	}
	dropped, err := counts.dropped.amountsOf(tx, crawlID)
	if err != nil {
		return Progress{}, fmt.Errorf("read the dropped page visits of crawl %s: %w", crawlID, err)
	}
	unadmittedLinks, err := counts.unadmittedLinks.Get(tx, crawlID)
	if err != nil {
		return Progress{}, fmt.Errorf("read the unadmitted links of crawl %s: %w", crawlID, err)
	}
	return Progress{
		Completed:       completed,
		Dropped:         dropped,
		UnadmittedLinks: int64(unadmittedLinks),
	}, nil
}

func (counts *Counts) ProgressByHostOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (map[string]Progress, error) {
	completedByHost, err := counts.completed.amountsByHostOf(tx, crawlID)
	if err != nil {
		return nil, fmt.Errorf("read the completed page visits of the hosts of crawl %s: %w",
			crawlID, err)
	}
	droppedByHost, err := counts.dropped.amountsByHostOf(tx, crawlID)
	if err != nil {
		return nil, fmt.Errorf("read the dropped page visits of the hosts of crawl %s: %w",
			crawlID, err)
	}
	return progressByHostFrom(completedByHost, droppedByHost), nil
}

func progressByHostFrom(
	completedByHost map[string]map[yacycrawlcontract.PageVisitResult]int64,
	droppedByHost map[string]map[yacycrawlcontract.PageVisitDropReason]int64,
) map[string]Progress {
	progressByHost := map[string]Progress{}
	for host, completed := range completedByHost {
		progress := progressOfHostIn(progressByHost, host)
		progress.Completed = completed
		progressByHost[host] = progress
	}
	for host, dropped := range droppedByHost {
		progress := progressOfHostIn(progressByHost, host)
		progress.Dropped = dropped
		progressByHost[host] = progress
	}
	return progressByHost
}

func progressOfHostIn(progressByHost map[string]Progress, host string) Progress {
	if progress, known := progressByHost[host]; known {
		return progress
	}
	return newProgress()
}

func (counts *Counts) ForgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := counts.completed.forgetCrawl(tx, crawlID); err != nil {
		return fmt.Errorf("forget the completed page visits of crawl %s: %w", crawlID, err)
	}
	if err := counts.dropped.forgetCrawl(tx, crawlID); err != nil {
		return fmt.Errorf("forget the dropped page visits of crawl %s: %w", crawlID, err)
	}
	if err := counts.unadmittedLinks.DeleteWithin(
		tx,
		crawlKeys.KeysWithFirst(crawlID),
	); err != nil {
		return fmt.Errorf("forget the unadmitted links of crawl %s: %w", crawlID, err)
	}
	return nil
}
