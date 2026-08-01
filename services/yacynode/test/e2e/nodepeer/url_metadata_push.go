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

	req := yacyproto.TransferURLRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Iam:         pushSenderHash,
		YouAre:      nodeHash,
		URLCount:    1,
		URLs:        []yacymodel.URLMetadata{metadata},
	}

	result := probe.PostRaw(
		ctx,
		nodeURL+yacyproto.PathTransferURL,
		req.Form().Encode(),
		"Content-Type: application/x-www-form-urlencoded",
	)
	if !result.OK {
		t.Fatalf("push url metadata to node failed: %s", result.Diag())
	}
}
