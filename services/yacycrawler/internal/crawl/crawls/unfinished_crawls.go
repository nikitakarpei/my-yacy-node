package crawls

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const unfinishedCrawlsBucket = "unfinished_crawls"

type unfinishedCrawls struct {
	crawlIndexes
	crawls *vault.Collection[crawlid.CrawlID, Crawl]
}

func registerUnfinishedCrawls(
	ctx context.Context,
	storage *vault.Vault,
) (unfinishedCrawls, error) {
	var unfinished unfinishedCrawls
	var err error
	if unfinished.crawlIndexes, err = registerCrawlIndexes(
		storage,
		unfinishedCrawlsBucket,
	); err != nil {
		return unfinishedCrawls{}, err
	}
	if unfinished.crawls, err = storage.RegisterProjectedCollection(
		ctx, unfinishedCrawlsBucket, crawlIDKeys, crawlCodec{},
		unfinished.projections(),
	); err != nil {
		return unfinishedCrawls{}, fmt.Errorf("register %s: %w", unfinishedCrawlsBucket, err)
	}
	return unfinished, nil
}

func (unfinished unfinishedCrawls) crawlOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (Crawl, bool, error) {
	return unfinished.crawls.Get(tx, crawlID)
}

func (unfinished unfinishedCrawls) crawlIDs(tx *vault.Txn) ([]crawlid.CrawlID, error) {
	var crawlIDs []crawlid.CrawlID
	err := unfinished.crawls.Scan(
		tx,
		vault.EveryKey(),
		func(crawlID crawlid.CrawlID, _ Crawl) (bool, error) {
			crawlIDs = append(crawlIDs, crawlID)
			return true, nil
		},
	)
	return crawlIDs, err
}

func (unfinished unfinishedCrawls) put(tx *vault.Txn, crawlID crawlid.CrawlID, crawl Crawl) error {
	_, err := unfinished.crawls.Put(tx, crawlID, crawl)
	return err
}

func (unfinished unfinishedCrawls) remove(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	_, err := unfinished.crawls.Delete(tx, crawlID)
	return err
}
