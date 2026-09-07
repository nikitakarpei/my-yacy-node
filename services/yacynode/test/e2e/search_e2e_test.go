//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/nodepeer"
)

const (
	searchYaCyAlias = "yacy-search-e2e"
	searchNodeAlias = "node-search-e2e"

	// searchTerm appears in no document YaCy holds, so a hit can only come from
	// the node over /yacy/search.html.
	searchTerm = "yacyrwinodesearchprobe"
)

var searchDocument = yacymodel.URLMetadata{
	Address:      "http://search-probe.example.com/node-only-document",
	Title:        "node only document",
	DocumentType: yacymodel.DocumentTypeHTML,
}

func TestRealYaCyFindsNodeDocument(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)

	egressproxy.Start(t, ctx, network.Name)

	_, yacyURL := yacypeer.Start(
		t,
		ctx,
		probe,
		network.Name,
		searchYaCyAlias,
		yacypeer.RemoteSearchOverrides()...,
	)

	nodeHash := mustHash("SRCHNODEHASH")
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       searchNodeAlias,
		Hash:        nodeHash,
		SeedlistURL: "http://" + searchYaCyAlias + ":" + peerclient.Port + "/yacy/seedlist.html",
	})

	documentHash, err := searchDocument.Hash()
	if err != nil {
		t.Fatalf("document hash: %v", err)
	}

	nodepeer.PushPosting(
		t,
		ctx,
		probe,
		nodeURL,
		nodeHash,
		yacymodel.WordHash(searchTerm),
		documentHash,
	)
	nodepeer.PushURLMetadata(t, ctx, probe, nodeURL, nodeHash, searchDocument)

	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)
	waitPeerIndexedWords(t, ctx, probe, yacyURL, nodeHash, 60*time.Second)

	found := pollwait.For(120*time.Second, func() bool {
		for _, link := range yacypeer.ResultLinksFor(t, ctx, probe, yacyURL, searchTerm) {
			if link == searchDocument.Address {
				return true
			}
		}
		return false
	})
	if !found {
		t.Fatalf(
			"real YaCy never returned %s for %q; it holds the document nowhere else",
			searchDocument.Address,
			searchTerm,
		)
	}
}
