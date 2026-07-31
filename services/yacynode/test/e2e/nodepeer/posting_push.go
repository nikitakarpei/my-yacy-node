//go:build e2e

package nodepeer

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

var pushSenderHash = mustHash("PUSHSENDER01")

func mustHash(raw string) yacymodel.Hash {
	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		panic(err)
	}

	return hash
}

// PushPosting delivers one RWI posting to the node under test over the same
// transferRWI wire call a peer would use, so the node schedules it for distribution.
func PushPosting(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	word yacymodel.Hash,
	docURL yacymodel.URLHash,
) {
	t.Helper()

	req := yacyproto.TransferRWIRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Iam:         pushSenderHash,
		YouAre:      nodeHash,
		WordCount:   1,
		EntryCount:  1,
		Indexes:     []yacymodel.RWIPosting{{WordHash: word, URLHash: docURL}},
	}

	result := probe.PostRaw(
		ctx,
		nodeURL+yacyproto.PathTransferRWI,
		req.Form().Encode(),
		"Content-Type: application/x-www-form-urlencoded",
	)
	if !result.OK {
		t.Fatalf("push posting to node failed: %s", result.Diag())
	}
}
