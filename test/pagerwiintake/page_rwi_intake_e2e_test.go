//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagerwiservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagescrapeservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	originAlias = "origin"
	originPage  = `<html lang="en"><title>Hi</title><body>words here</body></html>`
	nodeAlias   = "node"
	storeWait   = 2 * time.Minute
	nodeDurable = "yacy-node"
)

func TestScrapedPageReachesTheNodeAsAPageRWI(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)
	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := staticpage.Start(t, ctx, network.Name, originAlias, originPage)
	egressproxy.Start(t, ctx, network.Name)
	pagescrapeservice.Start(t, ctx, network.Name)
	pagerwiservice.Start(t, ctx, network.Name)
	nodeKey := peerkeytest.Key(0)
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName:    network.Name,
		Alias:          nodeAlias,
		Key:            nodeKey,
		PageRWINATSURL: natsjetstream.NetworkURL(),
	})
	awaitNodeConsumer(t, ctx, connectJetStream(t, natsURL))

	publishScrapeRequest(t, ctx, connectJetStream(t, natsURL), originURL)

	if !pollwait.For(storeWait, func() bool {
		stored, ok := peerclient.QueryCount(
			ctx, probe, nodeURL, nodeKey.Hash(), yacyproto.ObjectLURLCount,
		)
		return ok && stored == 1
	}) {
		t.Fatal("the node never stored the url of the scraped page")
	}
}

func connectJetStream(t *testing.T, natsURL string) jetstream.JetStream {
	t.Helper()
	conn, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(conn.Close)
	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func awaitNodeConsumer(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	if !pollwait.For(storeWait, func() bool {
		_, err := js.Consumer(ctx, pagerwicontract.PageRWIsStreamName, nodeDurable)
		return err == nil
	}) {
		t.Fatal("the node never opened its rwi consumer")
	}
}

func publishScrapeRequest(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	pageURL string,
) {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{
			PageURL: canonicalurltest.CanonicalURLOf(t, pageURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	if _, err := js.Publish(ctx, pagescrapecontract.ScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
	}
}
