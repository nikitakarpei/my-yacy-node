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

func TestMatchesContentKindStrict(t *testing.T) {
	if !matchesContentKind(postingOfType(yacymodel.DocumentTypeImage), imageContent, true) {
		t.Fatal("image document should match strict image")
	}
	if matchesContentKind(postingOfType(yacymodel.DocumentTypeAudio), imageContent, true) {
		t.Fatal("audio document should not match strict image")
	}
	if !matchesContentKind(postingOfType(yacymodel.DocumentTypeMovie), videoContent, true) {
		t.Fatal("movie document should match strict video")
	}
}

func TestMatchesContentKindNonStrict(t *testing.T) {
	if !matchesContentKind(postingWith(yacymodel.Appearance{HasAudio: true}), audioContent, false) {
		t.Fatal("audio appearance should match non-strict audio")
	}
	if matchesContentKind(postingWith(yacymodel.Appearance{HasImage: true}), audioContent, false) {
		t.Fatal("image appearance should not match non-strict audio")
	}
	if !matchesContentKind(postingWith(yacymodel.Appearance{HasVideo: true}), videoContent, false) {
		t.Fatal("video appearance should match non-strict video")
	}
	if !matchesContentKind(
		postingWith(yacymodel.Appearance{HasApp: true}),
		applicationContent,
		false,
	) {
		t.Fatal("app appearance should match app")
	}
}

func TestMatchesContentKindPassthrough(t *testing.T) {
	posting := postingOfType(yacymodel.DocumentTypeImage)
	if !matchesContentKind(posting, anyContent, false) {
		t.Fatal("any content kind should pass through")
	}
	if !matchesContentKind(posting, anyContent, true) {
		t.Fatal("any content kind should pass through when strict")
	}
}

func TestMatchesSiteHost(t *testing.T) {
	location, err := yacymodel.ParseURLHash("0123456789AB")
	if err != nil {
		t.Fatalf("parse url hash: %v", err)
	}
	if !matchesSiteHost(location, yacymodel.None[yacymodel.HostHash]()) {
		t.Fatal("empty site hash should match")
	}
	if !matchesSiteHost(location, yacymodel.Some(mustHostHash(t, "6789AB"))) {
		t.Fatal("matching host hash should match")
	}
	if matchesSiteHost(location, yacymodel.Some(mustHostHash(t, "000000"))) {
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

func TestMatchesRequiredProperties(t *testing.T) {
	posting := postingWith(yacymodel.Appearance{HasImage: true})

	if !matchesRequiredProperties(posting, yacymodel.None[yacymodel.Appearance]()) {
		t.Fatal("no required properties should match")
	}
	if !matchesRequiredProperties(
		posting,
		yacymodel.Some(yacymodel.Appearance{HasImage: true}),
	) {
		t.Fatal("required property present in appearance should match")
	}
	if matchesRequiredProperties(
		posting,
		yacymodel.Some(yacymodel.Appearance{HasVideo: true}),
	) {
		t.Fatal("required property absent from appearance should not match")
	}
}

func TestDocumentSet(t *testing.T) {
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
