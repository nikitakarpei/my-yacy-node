package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlprogress"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawls"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/frontier"
)

type crawlStateWiring struct {
	crawls   *crawls.Crawls
	frontier *frontier.Frontier
	progress *crawlprogress.Counts
}

func (wiring *crawlStateWiring) CrawlFinished(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return wiring.frontier.Close(tx, crawlID)
}

func (wiring *crawlStateWiring) CrawlOwesNothing(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return wiring.crawls.Settle(tx, crawlID)
}

func (wiring *crawlStateWiring) CrawlForgotten(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := wiring.frontier.ForgetCrawl(tx, crawlID); err != nil {
		return err
	}
	return wiring.progress.ForgetCrawl(tx, crawlID)
}
