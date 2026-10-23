package crawlprogress

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/vaultkeyparts"
)

type visitCounts[Kind ~string] struct {
	ofCrawl *vault.Amounts[crawlCount[Kind]]
	ofHost  *vault.Amounts[hostCount[Kind]]
}

type crawlCount[Kind ~string] struct {
	crawlID crawlid.CrawlID
	kind    Kind
}

type hostCount[Kind ~string] struct {
	crawlID crawlid.CrawlID
	host    string
	kind    Kind
}

var crawlKeys = vault.SingleKey(vaultkeyparts.CrawlID)

func registerVisitCounts[Kind ~string](
	storage *vault.Vault,
	crawlBucket vault.Name,
	hostBucket vault.Name,
) (visitCounts[Kind], error) {
	ofCrawl, err := storage.RegisterAmounts(crawlBucket, crawlCountKeys[Kind]())
	if err != nil {
		return visitCounts[Kind]{}, fmt.Errorf("register %s: %w", crawlBucket, err)
	}
	ofHost, err := storage.RegisterAmounts(hostBucket, hostCountKeys[Kind]())
	if err != nil {
		return visitCounts[Kind]{}, fmt.Errorf("register %s: %w", hostBucket, err)
	}
	return visitCounts[Kind]{ofCrawl: ofCrawl, ofHost: ofHost}, nil
}

func crawlCountKeys[Kind ~string]() vault.KeyLayout[crawlCount[Kind]] {
	return vault.PairKey(vaultkeyparts.CrawlID, kindKeyPart[Kind]()).KeyLayoutFor(
		func(key crawlCount[Kind]) (crawlid.CrawlID, Kind) { return key.crawlID, key.kind },
		func(crawlID crawlid.CrawlID, kind Kind) crawlCount[Kind] {
			return crawlCount[Kind]{crawlID: crawlID, kind: kind}
		},
	)
}

func kindKeyPart[Kind ~string]() vault.KeyPart[Kind] {
	return vault.TextKeyPartFrom(
		func(kind Kind) string { return string(kind) },
		func(text string) (Kind, error) { return Kind(text), nil },
	)
}

func hostCountKeys[Kind ~string]() vault.KeyLayout[hostCount[Kind]] {
	return vault.TripleKey(vaultkeyparts.CrawlID, vault.TextKeyPart, kindKeyPart[Kind]()).
		KeyLayoutFor(
			func(key hostCount[Kind]) (crawlid.CrawlID, string, Kind) {
				return key.crawlID, key.host, key.kind
			},
			func(crawlID crawlid.CrawlID, host string, kind Kind) hostCount[Kind] {
				return hostCount[Kind]{crawlID: crawlID, host: host, kind: kind}
			},
		)
}

func (counts visitCounts[Kind]) raise(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	kind Kind,
	by uint,
) error {
	if err := counts.ofCrawl.Raise(
		tx,
		crawlCount[Kind]{crawlID: crawlID, kind: kind},
		by,
	); err != nil {
		return err
	}
	return counts.ofHost.Raise(
		tx,
		hostCount[Kind]{crawlID: crawlID, host: host, kind: kind},
		by,
	)
}

func (counts visitCounts[Kind]) amountsOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (map[Kind]int64, error) {
	amounts := map[Kind]int64{}
	err := counts.ofCrawl.Scan(
		tx,
		crawlKeys.KeysWithFirst(crawlID),
		func(key crawlCount[Kind], amount int) (bool, error) {
			amounts[key.kind] = int64(amount)
			return true, nil
		},
	)
	return amounts, err
}

func (counts visitCounts[Kind]) amountsByHostOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (map[string]map[Kind]int64, error) {
	amountsByHost := map[string]map[Kind]int64{}
	err := counts.ofHost.Scan(
		tx,
		crawlKeys.KeysWithFirst(crawlID),
		func(key hostCount[Kind], amount int) (bool, error) {
			if amountsByHost[key.host] == nil {
				amountsByHost[key.host] = map[Kind]int64{}
			}
			amountsByHost[key.host][key.kind] = int64(amount)
			return true, nil
		},
	)
	return amountsByHost, err
}

func (counts visitCounts[Kind]) forgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := counts.ofCrawl.DeleteWithin(tx, crawlKeys.KeysWithFirst(crawlID)); err != nil {
		return err
	}
	return counts.ofHost.DeleteWithin(tx, crawlKeys.KeysWithFirst(crawlID))
}
