//go:build e2e

package nodepeer

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const postingsPerTransfer = 1000

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

	PushPostings(t, ctx, probe, nodeURL, nodeHash, word, []yacymodel.URLHash{docURL})
}

// PushPostings delivers the RWI postings of one word over every named document
// to the node under test, over as many transferRWI wire calls as the node admits,
// so the node can hold a word over more documents than it lists in an index abstract.
func PushPostings(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	word yacymodel.Hash,
	documents []yacymodel.URLHash,
) {
	t.Helper()

	for first := 0; first < len(documents); first += postingsPerTransfer {
		transferPostings(
			t, ctx, probe, nodeURL, nodeHash, word,
			documents[first:min(first+postingsPerTransfer, len(documents))],
		)
	}
}

func transferPostings(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	word yacymodel.Hash,
	documents []yacymodel.URLHash,
) {
	t.Helper()

	req := yacyproto.TransferRWIRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Iam:         pushSenderHash,
		YouAre:      nodeHash,
		WordCount:   1,
		EntryCount:  len(documents),
		Indexes:     postingsOf(t, word, documents),
	}

	result := probe.PostRaw(
		ctx,
		nodeURL+yacyproto.PathTransferRWI,
		req.Form().Encode(),
		"Content-Type: application/x-www-form-urlencoded",
	)
	if !result.OK {
		t.Fatalf("push %d postings to node failed: %s", len(documents), result.Diag())
	}
}

func postingsOf(
	t *testing.T,
	word yacymodel.Hash,
	documents []yacymodel.URLHash,
) []yacymodel.RWIPosting {
	t.Helper()

	language, err := yacymodel.ParseLanguage("en")
	if err != nil {
		t.Fatalf("posting language: %v", err)
	}

	postings := make([]yacymodel.RWIPosting, 0, len(documents))
	for _, document := range documents {
		postings = append(postings, yacymodel.RWIPosting{
			WordHash: word,
			URLHash:  document,
			Language: language,
		})
	}

	return postings
}
