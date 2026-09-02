//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
)

const (
	originAlias    = "origin"
	originBody     = "words here"
	originPage     = `<html lang="en"><title>Hi</title><body>` + originBody + `</body></html>`
	originPageURL  = "http://" + originAlias + "/"
	missingPageURL = "http://" + originAlias + "/missing"
)

func startOrigin(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	staticpage.Start(t, ctx, networkName, originAlias, originPage)
}
