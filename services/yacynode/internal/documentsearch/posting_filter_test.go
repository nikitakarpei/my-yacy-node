package documentsearch

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func postingOfType(kind yacymodel.DocumentType) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{DocumentType: kind}
}

func postingWith(appearance yacymodel.Appearance) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{Appearance: appearance}
}

func TestIsOfDocumentType(t *testing.T) {
	if !isOfDocumentType(postingOfType(yacymodel.DocumentTypeImage), imageContent) {
		t.Fatal("image document should match strict image")
	}
	if isOfDocumentType(postingOfType(yacymodel.DocumentTypeAudio), imageContent) {
		t.Fatal("audio document should not match strict image")
	}
	if !isOfDocumentType(postingOfType(yacymodel.DocumentTypeMovie), videoContent) {
		t.Fatal("movie document should match strict video")
	}
	if !isOfDocumentType(postingWith(yacymodel.Appearance{HasApp: true}), applicationContent) {
		t.Fatal("app appearance should match app")
	}
	if !isOfDocumentType(postingOfType(yacymodel.DocumentTypeImage), anyContent) {
		t.Fatal("any content kind should pass through")
	}
}

func TestAppearsAsContentKind(t *testing.T) {
	if !appearsAsContentKind(postingWith(yacymodel.Appearance{HasAudio: true}), audioContent) {
		t.Fatal("audio appearance should match loose audio")
	}
	if appearsAsContentKind(postingWith(yacymodel.Appearance{HasImage: true}), audioContent) {
		t.Fatal("image appearance should not match loose audio")
	}
	if !appearsAsContentKind(postingWith(yacymodel.Appearance{HasVideo: true}), videoContent) {
		t.Fatal("video appearance should match loose video")
	}
	if !appearsAsContentKind(postingWith(yacymodel.Appearance{HasImage: true}), imageContent) {
		t.Fatal("image appearance should match loose image")
	}
	if !appearsAsContentKind(postingWith(yacymodel.Appearance{HasApp: true}), applicationContent) {
		t.Fatal("app appearance should match app")
	}
	if !appearsAsContentKind(postingOfType(yacymodel.DocumentTypeImage), anyContent) {
		t.Fatal("any content kind should pass through")
	}
}

func TestIsFromRequestedSite(t *testing.T) {
	documentHash, err := yacymodel.ParseURLHash("0123456789AB")
	if err != nil {
		t.Fatalf("parse url hash: %v", err)
	}
	if !isFromRequestedSite(documentHash, yacymodel.None[yacymodel.HostHash]()) {
		t.Fatal("empty site hash should match")
	}
	if !isFromRequestedSite(documentHash, yacymodel.Some(mustHostHash(t, "6789AB"))) {
		t.Fatal("matching host hash should match")
	}
	if isFromRequestedSite(documentHash, yacymodel.Some(mustHostHash(t, "000000"))) {
		t.Fatal("non-matching host hash should not match")
	}
}

func mustHostHash(t *testing.T, s string) yacymodel.HostHash {
	t.Helper()
	hash, err := yacymodel.ParseHostHash(s)
	if err != nil {
		t.Fatalf("ParseHostHash(%q): %v", s, err)
	}

	return hash
}

func TestSharesRequiredAppearance(t *testing.T) {
	posting := postingWith(yacymodel.Appearance{HasImage: true})

	if !sharesRequiredAppearance(posting, yacymodel.None[yacymodel.Appearance]()) {
		t.Fatal("no required properties should match")
	}
	if !sharesRequiredAppearance(
		posting,
		yacymodel.Some(yacymodel.Appearance{HasImage: true}),
	) {
		t.Fatal("required property present in appearance should match")
	}
	if sharesRequiredAppearance(
		posting,
		yacymodel.Some(yacymodel.Appearance{HasVideo: true}),
	) {
		t.Fatal("required property absent from appearance should not match")
	}
}

func TestDocumentSetHoldsEveryDocument(t *testing.T) {
	if documentSet(nil) != nil {
		t.Fatal("nil input should return nil")
	}
	first, second := urlHashFor("url-a"), urlHashFor("url-b")
	set := documentSet([]yacymodel.URLHash{first, second})
	if _, ok := set[first]; !ok {
		t.Fatal("first identifier missing")
	}
	if _, ok := set[second]; !ok {
		t.Fatal("second identifier missing")
	}
}
