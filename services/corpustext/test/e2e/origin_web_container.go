//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
)

const (
	originAlias = "origin"
	originTitle = "Riverside Guide"
	originBody  = "wildflowers grow along riverside trails"
	stemmedTerm = "wildflower"
	originPage  = `<html lang="en"><title>` + originTitle + `</title><body>` + originBody + `</body></html>`
)

func startOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return staticpage.Start(t, ctx, networkName, originAlias, originPage)
}
