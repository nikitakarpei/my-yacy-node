//go:build e2e

package yacypeer

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const DHTMinLocalRWIs = 100

func WaitRWICount(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	yacyHash yacymodel.Hash,
	want int,
	timeout time.Duration,
) {
	t.Helper()
	last := -1
	if pollwait.For(timeout, func() bool {
		count, ok := peerclient.QueryCount(ctx, probe, yacyURL, yacyHash, yacyproto.ObjectRWICount)
		if !ok {
			return false
		}
		last = count
		return count >= want
	}) {
		return
	}
	t.Fatalf(
		"YaCy never reported at least %d RWIs (last=%d)",
		want,
		last,
	)
}
