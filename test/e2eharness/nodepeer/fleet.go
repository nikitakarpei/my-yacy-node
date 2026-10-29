//go:build e2e

package nodepeer

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
)

type Peer struct {
	Alias string
	Hash  yacymodel.Hash
	URL   string
}

func KeyAfterFleet(index int) yacymodel.PeerKey {
	return peerkeytest.Key(FleetSize + index)
}

func StartFleet(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName, seedlistURL string,
) []Peer {
	t.Helper()
	fleet := make([]Peer, FleetSize)
	for i := range fleet {
		alias := fmt.Sprintf("node-tr-%02d", i)
		key := peerkeytest.Key(i)
		_, url := Start(t, ctx, probe, Config{
			NetworkName: networkName,
			Alias:       alias,
			Key:         key,
			SeedlistURL: seedlistURL,
		})
		fleet[i] = Peer{Alias: alias, Hash: key.Hash(), URL: url}
	}
	return fleet
}
