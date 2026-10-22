// Package crawlprogress counts the progress of each crawl and of each of its hosts: the
// visits it completed by result, the visits it dropped by reason, and the links its profile
// did not admit. A count can miss a visit or count it twice; it is not part of the visit.
package crawlprogress

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	completedCountPrefix = "completed/"
	droppedCountPrefix   = "dropped/"
	unadmittedLinksCount = "unadmitted-links"
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

func (progress Progress) withCount(count string, amount int) Progress {
	if result, completed := strings.CutPrefix(count, completedCountPrefix); completed {
		progress.Completed[yacycrawlcontract.PageVisitResult(result)] += int64(amount)
	}
	if reason, dropped := strings.CutPrefix(count, droppedCountPrefix); dropped {
		progress.Dropped[yacycrawlcontract.PageVisitDropReason(reason)] += int64(amount)
	}
	if count == unadmittedLinksCount {
		progress.UnadmittedLinks += int64(amount)
	}
	return progress
}

func (progress Progress) AmountOfURLs(owed int64) int64 {
	amount := owed
	for _, completed := range progress.Completed {
		amount += completed
	}
	for _, dropped := range progress.Dropped {
		amount += dropped
	}
	return amount
}

func completedCountOf(result yacycrawlcontract.PageVisitResult) string {
	return completedCountPrefix + string(result)
}

func droppedCountOf(reason yacycrawlcontract.PageVisitDropReason) string {
	return droppedCountPrefix + string(reason)
}

type Counts struct {
	ofCrawl *vault.Amounts[crawlCount]
	ofHost  *vault.Amounts[hostCount]
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
	return counts.raiseOfHost(tx, crawlID, host, completedCountOf(result), 1)
}

func (counts *Counts) CountDropped(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	reason yacycrawlcontract.PageVisitDropReason,
	amount uint,
) error {
	return counts.raiseOfHost(tx, crawlID, host, droppedCountOf(reason), amount)
}

func (counts *Counts) raiseOfHost(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	count string,
	by uint,
) error {
	if err := counts.ofCrawl.Raise(tx, crawlCount{crawlID: crawlID, count: count}, by); err != nil {
		return fmt.Errorf("count %s of crawl %s: %w", count, crawlID, err)
	}
	if err := counts.ofHost.Raise(
		tx,
		hostCount{crawlID: crawlID, host: host, count: count},
		by,
	); err != nil {
		return fmt.Errorf("count %s of host %s: %w", count, host, err)
	}
	return nil
}

func (counts *Counts) CountUnadmittedLinks(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	amount uint,
) error {
	if err := counts.ofCrawl.Raise(
		tx, crawlCount{crawlID: crawlID, count: unadmittedLinksCount}, amount,
	); err != nil {
		return fmt.Errorf("count unadmitted links of crawl %s: %w", crawlID, err)
	}
	return nil
}

func (counts *Counts) ProgressOf(tx *vault.Txn, crawlID crawlid.CrawlID) (Progress, error) {
	progress := newProgress()
	if err := counts.ofCrawl.Scan(
		tx,
		crawlCountKeyParts.KeysWithFirst(crawlID),
		func(key crawlCount, amount int) (bool, error) {
			progress = progress.withCount(key.count, amount)
			return true, nil
		},
	); err != nil {
		return Progress{}, fmt.Errorf("read the progress of crawl %s: %w", crawlID, err)
	}
	return progress, nil
}

func (counts *Counts) ProgressOfHostsOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (map[string]Progress, error) {
	progressOfHosts := map[string]Progress{}
	if err := counts.ofHost.Scan(
		tx,
		hostCountKeyParts.KeysWithFirst(crawlID),
		func(key hostCount, amount int) (bool, error) {
			progress, known := progressOfHosts[key.host]
			if !known {
				progress = newProgress()
			}
			progressOfHosts[key.host] = progress.withCount(key.count, amount)
			return true, nil
		},
	); err != nil {
		return nil, fmt.Errorf("read the progress of the hosts of crawl %s: %w", crawlID, err)
	}
	return progressOfHosts, nil
}

func (counts *Counts) ForgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := counts.ofCrawl.DeleteWithin(
		tx,
		crawlCountKeyParts.KeysWithFirst(crawlID),
	); err != nil {
		return fmt.Errorf("forget the counts of crawl %s: %w", crawlID, err)
	}
	if err := counts.ofHost.DeleteWithin(tx, hostCountKeyParts.KeysWithFirst(crawlID)); err != nil {
		return fmt.Errorf("forget the host counts of crawl %s: %w", crawlID, err)
	}
	return nil
}
