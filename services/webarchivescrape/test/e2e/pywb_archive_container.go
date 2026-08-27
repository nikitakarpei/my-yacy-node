//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pywbarchive"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/warcarchive"
)

func startPywbArchive(
	t *testing.T,
	ctx context.Context,
	networkName string,
	captures []warcarchive.Capture,
) pywbarchive.Archive {
	t.Helper()
	archive := warcarchive.Write(t, captures)
	return pywbarchive.Start(t, ctx, networkName, archive.Content())
}
