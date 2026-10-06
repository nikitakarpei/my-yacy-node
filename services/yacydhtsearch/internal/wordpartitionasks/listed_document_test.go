package wordpartitionasks_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestADocumentListedByItsHashCarriesNoMetadataAndNoPosting(t *testing.T) {
	t.Parallel()

	listedDocument := wordpartitionasks.ListedDocument{
		Hash: documentHashOf(t, "https://a.example/"),
	}

	if listedDocument.Metadata().Present() || listedDocument.Posting().Present() {
		t.Fatalf("the document listed by its hash carries %+v and %+v, want neither",
			listedDocument.Metadata(), listedDocument.Posting())
	}
}

func TestADocumentListedFromItsMetadataCarriesItAndThePostingSent(t *testing.T) {
	t.Parallel()

	metadata := yacymodel.URLMetadata{
		Hash:  documentHashOf(t, "https://a.example/"),
		Title: "Berlin",
	}
	posting := yacymodel.RWIPosting{URLHash: metadata.Hash, Hits: 3}

	listedDocument := wordpartitionasks.ListedDocumentFrom(metadata, yacymodel.Some(posting))

	carriedMetadata, metadataCarried := listedDocument.Metadata().Get()
	carriedPosting, postingCarried := listedDocument.Posting().Get()
	if listedDocument.Hash != metadata.Hash || !metadataCarried ||
		!reflect.DeepEqual(carriedMetadata, metadata) ||
		!postingCarried ||
		carriedPosting != posting {
		t.Fatalf("the listed document is %v with %+v and %+v, want %v with %+v and %+v",
			listedDocument.Hash, listedDocument.Metadata(), listedDocument.Posting(),
			metadata.Hash, metadata, posting)
	}
}

func TestADocumentListedFromMetadataWithoutAPostingCarriesNoPosting(t *testing.T) {
	t.Parallel()

	listedDocument := wordpartitionasks.ListedDocumentFrom(
		yacymodel.URLMetadata{Hash: documentHashOf(t, "https://a.example/")},
		yacymodel.None[yacymodel.RWIPosting](),
	)

	if listedDocument.Posting().Present() {
		t.Fatalf("the listed document carries the posting %+v, want none", listedDocument.Posting())
	}
}

func documentHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("hash of %s: %v", address, err)
	}

	return hash
}
