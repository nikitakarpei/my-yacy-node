//go:build e2e

package nodepeer

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

// PushURLMetadata delivers one URL metadata row to the node under test over the
// same transferURL wire call a peer would use, so the node can redistribute the
// postings that name that URL.
func PushURLMetadata(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	metadata yacymodel.URLMetadata,
) {
	t.Helper()

	PushURLMetadataRows(t, ctx, probe, nodeURL, nodeHash, []yacymodel.URLMetadata{metadata})
}

// PushURLMetadataRows delivers every named URL metadata row to the node under
// test in one transferURL wire call, so the node can describe a batch of
// documents it holds postings for.
func PushURLMetadataRows(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	rows []yacymodel.URLMetadata,
) {
	t.Helper()

	req := yacyproto.TransferURLRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Iam:         pushSenderHash,
		YouAre:      nodeHash,
		URLCount:    len(rows),
		URLs:        rows,
	}

	result := probe.PostRaw(
		ctx,
		nodeURL+yacyproto.PathTransferURL,
		req.Form().Encode(),
		"Content-Type: application/x-www-form-urlencoded",
	)
	if !result.OK {
		t.Fatalf("push %d url metadata rows to node failed: %s", len(rows), result.Diag())
	}
}
