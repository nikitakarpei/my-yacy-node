//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
)

const (
	originAlias     = "origin"
	originPage      = `<html lang="en"><title>Hi</title><body>words here</body></html>`
	disallowedPage  = "private.html"
	originRobotsTxt = "User-agent: yacycrawler\nDisallow: /private\n"
)

func startOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return staticpage.Start(t, ctx, networkName, originAlias, originPage)
}

func startOriginWithRobotsTxt(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return staticpage.StartSite(t, ctx, networkName, originAlias, map[string]string{
		"index.html":   originPage,
		disallowedPage: originPage,
		"robots.txt":   originRobotsTxt,
	})
}
