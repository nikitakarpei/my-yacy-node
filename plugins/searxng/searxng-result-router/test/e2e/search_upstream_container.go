//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
)

const (
	searchUpstreamAlias = "search-upstream"
	searchUpstreamPage  = `<html lang="en"><title>Search upstream</title><body>search upstream page</body></html>`
)

func startSearchUpstream(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	staticpage.Start(t, ctx, networkName, searchUpstreamAlias, searchUpstreamPage)
}

func searchUpstreamNetworkURL() string {
	return "http://" + searchUpstreamAlias + "/"
}
