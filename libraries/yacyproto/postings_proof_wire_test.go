package yacyproto_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestTransferRWIRequestCarriesPostingsProofsThatLeadToTheRoot(t *testing.T) {
	t.Parallel()

	crawled, offered := sampleProvenPagePostings(t, "url-a")
	req := yacyproto.TransferRWIRequest{
		Indexes:        offered.Postings(),
		PostingsProofs: postingsProofsOf(t, offered),
	}

	got, err := yacyproto.ParseTransferRWIRequest(context.Background(), req.Form())
	if err != nil {
		t.Fatalf("ParseTransferRWIRequest: %v", err)
	}

	received, err := yacymodel.ProvenPagePostingsFrom(
		got.Indexes,
		got.PostingsProofs[offered.URL()],
	)
	if err != nil {
		t.Fatal(err)
	}
	if !crawled.Root().Present() || received.Root() != crawled.Root() {
		t.Fatalf("received root = %v, want %v", received.Root(), crawled.Root())
	}
}

func TestTransferRWIRequestWithoutPostingsProofsWritesNoPostingsProofsField(t *testing.T) {
	t.Parallel()

	form := yacyproto.TransferRWIRequest{
		Indexes: []yacymodel.RWIPosting{sampleRWIPosting(t, "alpha", "url-a")},
	}.Form()

	if _, written := form[yacyproto.FieldPostingsProofs]; written {
		t.Fatalf("%s written without proofs", yacyproto.FieldPostingsProofs)
	}
}

func TestParseTransferRWIRequestDropsMalformedPostingsProofLines(t *testing.T) {
	t.Parallel()

	_, offered := sampleProvenPagePostings(t, "url-a")
	form := yacyproto.TransferRWIRequest{
		Indexes:        offered.Postings(),
		PostingsProofs: postingsProofsOf(t, offered),
	}.Form()
	malformedLines := []string{
		"no-separator",
		"short=AQQ",
		offered.URL().String() + "=not base64!",
		offered.URL().String() + "=AA",
	}
	form.Set(
		yacyproto.FieldPostingsProofs,
		strings.Join(malformedLines, "\n")+"\n"+form.Get(yacyproto.FieldPostingsProofs),
	)

	req, err := yacyproto.ParseTransferRWIRequest(context.Background(), form)
	if err != nil {
		t.Fatalf("ParseTransferRWIRequest: %v", err)
	}
	if _, kept := req.PostingsProofs[offered.URL()]; len(req.PostingsProofs) != 1 ||
		!kept {
		t.Fatalf("PostingsProofs = %v, want only the well-formed one", req.PostingsProofs)
	}
}

func TestParseTransferRWIRequestKeepsOneProofForEachOfferedURL(t *testing.T) {
	t.Parallel()

	_, firstOffered := sampleProvenPagePostings(t, "url-a")
	_, secondOffered := sampleProvenPagePostings(t, "url-b")
	_, notOffered := sampleProvenPagePostings(t, "url-c")
	form := yacyproto.TransferRWIRequest{
		Indexes: slices.Concat(firstOffered.Postings(), secondOffered.Postings()),
	}.Form()
	form.Set(yacyproto.FieldPostingsProofs, strings.Join([]string{
		postingsProofLineOf(t, notOffered),
		postingsProofLineOf(t, firstOffered),
		postingsProofLineOf(t, firstOffered),
		postingsProofLineOf(t, secondOffered),
		postingsProofLineOf(t, notOffered),
		postingsProofLineOf(t, secondOffered),
	}, "\n"))

	req, err := yacyproto.ParseTransferRWIRequest(context.Background(), form)
	if err != nil {
		t.Fatalf("ParseTransferRWIRequest: %v", err)
	}
	_, firstKept := req.PostingsProofs[firstOffered.Postings()[0].URLHash]
	_, secondKept := req.PostingsProofs[secondOffered.Postings()[0].URLHash]
	if len(req.PostingsProofs) != 2 || !firstKept || !secondKept {
		t.Fatalf("PostingsProofs = %v, want one for each offered url", req.PostingsProofs)
	}
}

func postingsProofLineOf(tb testing.TB, offered yacymodel.PagePostings) string {
	tb.Helper()

	return yacyproto.TransferRWIRequest{
		PostingsProofs: postingsProofsOf(tb, offered),
	}.Form().Get(yacyproto.FieldPostingsProofs)
}

func sampleProvenPagePostings(
	tb testing.TB,
	urlWord string,
) (crawled, offered yacymodel.PagePostings) {
	tb.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		tb.Fatalf("partitions: %v", err)
	}
	crawled, err = yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{
			sampleRWIPosting(tb, "alpha", urlWord),
			sampleRWIPosting(tb, "beta", urlWord),
			sampleRWIPosting(tb, "gamma", urlWord),
		},
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		tb.Fatal(err)
	}
	offered, narrowed := crawled.NarrowedTo(crawled.Postings()[:1]).Get()
	if !narrowed {
		tb.Fatal("no narrowing to the first posting")
	}

	return crawled, offered
}

func postingsProofsOf(
	tb testing.TB,
	offered yacymodel.PagePostings,
) map[yacymodel.URLHash]yacymodel.PostingsProof {
	tb.Helper()

	proof, proven := offered.Proof().Get()
	if !proven {
		tb.Fatal("offered postings carry no proof")
	}

	return map[yacymodel.URLHash]yacymodel.PostingsProof{offered.URL(): proof}
}
