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

// PushPosting delivers the RWI posting of one word in a document to the node
// under test over the same transferRWI wire call a peer would use, written from
// the document's URL metadata as one crawl writes both.
func PushPosting(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	nodeURL string,
	nodeHash yacymodel.Hash,
	word yacymodel.Hash,
	document yacymodel.URLMetadata,
) {
	t.Helper()

	PushPostings(t, ctx, probe, nodeURL, nodeHash, word, []yacymodel.URLMetadata{document})
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
	documents []yacymodel.URLMetadata,
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
	documents []yacymodel.URLMetadata,
) {
	t.Helper()

	req := yacyproto.TransferRWIRequest{
		NetworkName: yacyproto.DefaultNetwork,
		Iam:         pushSenderHash,
		YouAre:      nodeHash,
		WordCount:   1,
		EntryCount:  len(documents),
		Indexes:     postingsOf(word, documents),
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

func postingsOf(word yacymodel.Hash, documents []yacymodel.URLMetadata) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(documents))
	for _, document := range documents {
		postings = append(postings, postingOf(word, document))
	}

	return postings
}

func postingOf(word yacymodel.Hash, document yacymodel.URLMetadata) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:      word,
		URLHash:       document.Hash,
		Language:      document.Language.OrElse(yacymodel.LanguageOfUndeclaredDocument),
		DocumentType:  document.DocumentType,
		LastModified:  modifiedDayOf(document),
		TitleWords:    yacymodel.TitleWordsOf(document.Title),
		LocalLinks:    yacymodel.WrappedCardinalFrom(document.LocalLinks),
		ExternalLinks: yacymodel.WrappedCardinalFrom(document.ExternalLinks),
	}
}

func modifiedDayOf(document yacymodel.URLMetadata) yacymodel.MicroDate {
	for _, day := range []yacymodel.Optional[yacymodel.CalendarDay]{document.Modified, document.Loaded} {
		if present, ok := day.Get(); ok {
			if date, inRange := yacymodel.MicroDateOf(present.Time()).Get(); inRange {
				return date
			}
		}
	}

	return 0
}
