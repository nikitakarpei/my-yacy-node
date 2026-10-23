package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlprogress"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
)

type crawlStateWiring struct {
	frontier *frontier.Frontier
	progress *crawlprogress.Counts
}

func (wiring *crawlStateWiring) CrawlPaused(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return wiring.frontier.PauseQueueOf(tx, crawlID)
}

func (wiring *crawlStateWiring) CrawlResumed(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return wiring.frontier.ResumeQueueOf(tx, crawlID)
}

func (wiring *crawlStateWiring) CrawlCancelling(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return wiring.frontier.DrainQueueOf(tx, crawlID)
}

func (wiring *crawlStateWiring) PageVisitsDrained(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	host string,
	amount uint,
) error {
	return wiring.progress.CountDropped(
		tx, crawlID, host, yacycrawlcontract.DropReasonCrawlCancelled, amount,
	)
}

func (wiring *crawlStateWiring) CrawlForgotten(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := wiring.frontier.ForgetQueueOf(tx, crawlID); err != nil {
		return err
	}
	return wiring.progress.ForgetCrawl(tx, crawlID)
}
