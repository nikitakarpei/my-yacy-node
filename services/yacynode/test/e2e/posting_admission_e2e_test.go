//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const admissionNodeAlias = "node-admission-e2e"

var linkHeavyDocument = yacymodel.URLMetadata{
	Hash:          mustURLHash("http://example.invalid/link-heavy"),
	Address:       "http://example.invalid/link-heavy",
	Title:         "link heavy page",
	DocumentType:  yacymodel.DocumentTypeHTML,
	LocalLinks:    300,
	ExternalLinks: 700,
}

func TestNodeAdmitsThePostingOfAPageWithMoreLinksThanAPostingByteHolds(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)

	egressproxy.Start(t, ctx, network.Name)

	nodeKey := postingAdmissionNodeKey
	nodeHash := nodeKey.Hash()
	_, nodeURL := nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName: network.Name,
		Alias:       admissionNodeAlias,
		Key:         nodeKey,
	})

	nodepeer.PushURLMetadata(t, ctx, probe, nodeURL, nodeHash, linkHeavyDocument)
	nodepeer.PushPosting(
		t,
		ctx,
		probe,
		nodeURL,
		nodeHash,
		mustHash("LINKWORDHASH"),
		linkHeavyDocument,
	)

	if !pollwait.For(10*time.Second, func() bool {
		count, ok := peerclient.QueryCount(ctx, probe, nodeURL, nodeHash, yacyproto.ObjectRWICount)

		return ok && count > 0
	}) {
		t.Fatal(
			"node never admitted the posting of a page with more links than a posting byte holds",
		)
	}
}
