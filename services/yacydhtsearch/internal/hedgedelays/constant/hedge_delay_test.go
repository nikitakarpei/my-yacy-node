package constant_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/hedgedelays/constant"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestEveryPeerHedgesAfterTheConfiguredDelay(t *testing.T) {
	t.Parallel()

	configuredDelay := 250 * time.Millisecond

	hedgeDelayOfPeer := constant.New(configuredDelay).HedgeDelayOf(
		t.Context(),
		peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash("peer-0"),
			Address: "http://10.0.0.1:8090",
		},
	)

	if hedgeDelayOfPeer != configuredDelay {
		t.Fatalf("the peer hedges after %s, want %s", hedgeDelayOfPeer, configuredDelay)
	}
}
