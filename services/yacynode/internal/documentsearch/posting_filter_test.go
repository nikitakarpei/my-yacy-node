package documentsearch

import (
	"context"
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
	ctx := context.Background()
	const location = yacymodel.URLHash("0123456789AB")
	if !matchesSiteHost(ctx, location, "") {
		t.Fatal("empty site hash should match")
	}
	if !matchesSiteHost(ctx, location, "6789AB") {
		t.Fatal("matching host hash should match")
	}
	if matchesSiteHost(ctx, location, "000000") {
		t.Fatal("non-matching host hash should not match")
	}
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

func TestRequiredPropertiesNoOp(t *testing.T) {
	empty, err := requiredProperties("")
	if err != nil {
		t.Fatalf("requiredProperties: %v", err)
	}
	if empty.Present() {
		t.Fatal("empty required properties should be a no-op")
	}

	allOn, err := requiredProperties(yacymodel.Encode([]byte{0xff, 0xff, 0xff, 0xff}))
	if err != nil {
		t.Fatalf("requiredProperties: %v", err)
	}
	if allOn.Present() {
		t.Fatal("all-on required properties should be a no-op")
	}
}

func TestRequiredPropertiesRejectsMalformed(t *testing.T) {
	if _, err := requiredProperties("@@not-base64@@"); err == nil {
		t.Fatal("malformed required properties should fail")
	}
}

func TestRequiredPropertiesDecodesAppearance(t *testing.T) {
	encoded := yacymodel.Encode(yacymodel.Appearance{HasImage: true}.Bitfield())

	required, err := requiredProperties(encoded)
	if err != nil {
		t.Fatalf("requiredProperties: %v", err)
	}
	traits, ok := required.Get()
	if !ok {
		t.Fatal("single-flag required properties should be present")
	}
	if !traits.HasImage {
		t.Errorf("decoded appearance = %+v, want HasImage", traits)
	}
}

func TestDocumentSet(t *testing.T) {
	if documentSet(nil) != nil {
		t.Fatal("nil input should return nil")
	}
	first, second := hashFor("url-a"), hashFor("url-b")
	set := documentSet([]yacymodel.Hash{first, second})
	if _, ok := set[first]; !ok {
		t.Fatal("first identifier missing")
	}
	if _, ok := set[second]; !ok {
		t.Fatal("second identifier missing")
	}
}
