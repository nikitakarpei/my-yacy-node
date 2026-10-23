package crawls

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const finishedCrawlsBucket = "finished_crawls"

type finishedCrawls struct {
	crawlIndexes
	crawls *vault.ExpiringCollection[crawlid.CrawlID, Crawl]
}

func registerFinishedCrawls(
	ctx context.Context,
	storage *vault.Vault,
	retention time.Duration,
	forgotten ForgottenCrawls,
) (finishedCrawls, error) {
	var finished finishedCrawls
	var err error
	if finished.crawlIndexes, err = registerCrawlIndexes(
		storage,
		finishedCrawlsBucket,
	); err != nil {
		return finishedCrawls{}, err
	}
	if finished.crawls, err = storage.RegisterExpiringCollection(
		ctx, finishedCrawlsBucket, crawlIDKeys, crawlCodec{}, retention,
		finished.projectedWith(), forgotten.CrawlForgotten,
	); err != nil {
		return finishedCrawls{}, fmt.Errorf("register %s: %w", finishedCrawlsBucket, err)
	}
	return finished, nil
}

func (finished finishedCrawls) crawlOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (Crawl, bool, error) {
	return finished.crawls.Get(tx, crawlID)
}

func (finished finishedCrawls) put(tx *vault.Txn, crawlID crawlid.CrawlID, crawl Crawl) error {
	return finished.crawls.Put(tx, crawlID, crawl)
}
