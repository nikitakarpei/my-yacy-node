//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/scraperequeststream"
)

func startWebResearchStack(t *testing.T, ctx context.Context) string {
	t.Helper()
	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	scraperequeststream.Provision(t, ctx, natsURL)
	startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	startSearXNG(t, ctx, network.Name)

	return startWebResearchMCP(t, ctx, network.Name)
}
