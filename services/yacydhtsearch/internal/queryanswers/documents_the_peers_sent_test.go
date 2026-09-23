package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func metadataOfDocumentAt(t *testing.T, address string, title string) yacymodel.URLMetadata {
	t.Helper()

	return yacymodel.URLMetadata{Hash: documentOf(t, address), Address: address, Title: title}
}

func TestDocumentsThePeersSentComeBackInOrderTheyWereSentIn(t *testing.T) {
	t.Parallel()

	documents := queryanswers.EmptyDocumentsThePeersSent()
	documents.KeepMetadataThePeerSent(
		metadataOfDocumentAt(t, "https://first.example/", "First"),
		yacymodel.WordHash("a peer"),
	)
	documents.KeepMetadataThePeerSent(
		metadataOfDocumentAt(t, "https://second.example/", "Second"),
		yacymodel.WordHash("another peer"),
	)

	foundDocuments := documents.FoundDocuments()

	if len(foundDocuments) != 2 ||
		foundDocuments[0].Address != "https://first.example/" ||
		foundDocuments[1].Address != "https://second.example/" {
		t.Fatalf(
			"the documents come back as %+v, want the first one and then the second one",
			foundDocuments,
		)
	}
}

func TestDocumentASecondPeerSentMetadataOfHoldsTwoMetadataReplicas(t *testing.T) {
	t.Parallel()

	documents := queryanswers.EmptyDocumentsThePeersSent()
	documents.KeepMetadataThePeerSent(
		metadataOfDocumentAt(t, "https://shared.example/", "As the first peer holds it"),
		yacymodel.WordHash("the first peer"),
	)
	documents.KeepMetadataThePeerSent(
		metadataOfDocumentAt(t, "https://shared.example/", "As the second peer holds it"),
		yacymodel.WordHash("the second peer"),
	)

	foundDocuments := documents.FoundDocuments()

	if len(foundDocuments) != 1 {
		t.Fatalf("the documents come back as %+v, want the one document once", foundDocuments)
	}
	metadataReplicas := foundDocuments[0].MetadataReplicas
	if len(metadataReplicas) != 2 ||
		metadataReplicas[0].Holder != yacymodel.WordHash("the first peer") ||
		metadataReplicas[1].Holder != yacymodel.WordHash("the second peer") ||
		foundDocuments[0].Title != "As the first peer holds it" {
		t.Fatalf(
			"the document holds %+v under the title %q, want the metadata of both peers under "+
				"the title of the first one",
			metadataReplicas,
			foundDocuments[0].Title,
		)
	}
}

func TestPostingsOfDocumentComeBackOnThatDocumentAlone(t *testing.T) {
	t.Parallel()

	documents := queryanswers.EmptyDocumentsThePeersSent()
	documents.KeepDocumentThePeerMatched(
		yacymodel.WordHash("the first peer"),
		yacymodel.WordHash(countedWord),
		metadataOfDocumentAt(t, "https://counted.example/", "Counted"),
		yacymodel.Some(yacymodel.RWIPosting{Hits: 7}),
	)
	documents.KeepDocumentThePeerMatched(
		yacymodel.WordHash("the first peer"),
		yacymodel.WordHash(countedWord),
		metadataOfDocumentAt(t, "https://uncounted.example/", "Uncounted"),
		yacymodel.None[yacymodel.RWIPosting](),
	)

	foundDocuments := documents.FoundDocuments()

	postingReplicas := foundDocuments[0].PostingReplicas
	if len(postingReplicas) != 1 ||
		postingReplicas[0].Holder != yacymodel.WordHash("the first peer") ||
		postingReplicas[0].Posting.Hits != 7 ||
		len(foundDocuments[1].PostingReplicas) != 0 {
		t.Fatalf(
			"the documents hold the postings %+v and %+v, want the one posting on the document "+
				"it was sent for",
			postingReplicas,
			foundDocuments[1].PostingReplicas,
		)
	}
}

func TestDocumentsNoPeerSentAnythingForComeBackAsNone(t *testing.T) {
	t.Parallel()

	foundDocuments := queryanswers.EmptyDocumentsThePeersSent().FoundDocuments()

	if foundDocuments != nil {
		t.Fatalf("the documents come back as %+v, want none", foundDocuments)
	}
}
