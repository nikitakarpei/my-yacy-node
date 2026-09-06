//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerdirectory"
)

func waitPeerActiveConnected(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	peerHash yacymodel.Hash,
	timeout time.Duration,
) {
	t.Helper()
	if pollwait.For(timeout, func() bool {
		result := probe.Get(ctx, yacyURL+"/Network.xml?page=1&maxCount=1000")
		if !result.OK {
			return false
		}
		active, err := peerdirectory.ActivePeerHashes([]byte(result.Body))
		if err != nil {
			return false
		}
		_, ok := active[peerHash.String()]
		return ok
	}) {
		return
	}
	t.Fatalf("YaCy never saw peer hash %s as an active connected peer", peerHash)
}

func waitConnectedFleetPeers(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	fleet []nodepeer.Peer,
	want int,
	timeout time.Duration,
) {
	t.Helper()
	hashes := make(map[string]struct{}, len(fleet))
	for _, node := range fleet {
		hashes[node.Hash.String()] = struct{}{}
	}
	most := 0
	if pollwait.For(timeout, func() bool {
		result := probe.Get(ctx, yacyURL+"/Network.xml?page=1&maxCount=1000")
		if !result.OK {
			return false
		}
		active, err := peerdirectory.ActivePeerHashes([]byte(result.Body))
		if err != nil {
			return false
		}
		connected := 0
		for hash := range active {
			if _, ok := hashes[hash]; ok {
				connected++
			}
		}
		if connected > most {
			most = connected
		}
		return connected >= want
	}) {
		return
	}
	t.Fatalf(
		"YaCy connected at most %d of %d fleet peers, short of the %d it needs to distribute RWIs",
		most,
		len(fleet),
		want,
	)
}

// waitPeerIndexedWords waits until YaCy publishes a non-zero ICount for the
// peer. Below one indexed word YaCy drops the peer from a remote search
// (DHTSelection.java:215).
func waitPeerIndexedWords(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	peerHash yacymodel.Hash,
	timeout time.Duration,
) {
	t.Helper()
	if pollwait.For(timeout, func() bool {
		result := probe.Get(ctx, yacyURL+"/yacy/seedlist.xml")
		if !result.OK {
			return false
		}
		counts, err := peerdirectory.IndexedWordCounts([]byte(result.Body))
		if err != nil {
			return false
		}
		return counts[peerHash.String()] > 0
	}) {
		return
	}
	if result := probe.Get(ctx, yacyURL+"/yacy/seedlist.xml"); result.OK {
		t.Logf("final seedlist.xml:\n%s", result.Body)
	}
	t.Fatalf("YaCy never saw a non-zero ICount for peer hash %s", peerHash)
}
