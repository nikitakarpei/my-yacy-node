package postingidentity_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
)

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func TestKeyIsStableAndDistinctPerPosting(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	posting := postingidentity.IdentityOf(word, url)
	keyOf := postingidentity.KeyLayout.Encode

	if !bytes.Equal(keyOf(posting).Bytes(), keyOf(postingidentity.IdentityOf(word, url)).Bytes()) {
		t.Fatal("one posting addresses two rows")
	}
	for name, other := range map[string]postingidentity.Identity{
		"another word": postingidentity.IdentityOf(yacymodel.WordHash("w2"), url),
		"another url":  postingidentity.IdentityOf(word, urlHash("u2")),
	} {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(keyOf(posting).Bytes(), keyOf(other).Bytes()) {
				t.Fatal("two postings share one row")
			}
		})
	}
}

func TestKeyRoundTripsToTheSamePosting(t *testing.T) {
	posting := postingidentity.IdentityOf(yacymodel.WordHash("w1"), urlHash("u1"))

	encoded := postingidentity.KeyLayout.Encode(posting).Bytes()
	decoded, err := postingidentity.KeyLayout.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded != posting {
		t.Fatalf("Decode = %+v, want %+v", decoded, posting)
	}
}

func TestIdentityOfCarriesWordAndURL(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	identity := postingidentity.IdentityOf(word, url)

	if identity.Word != word || identity.URL != url {
		t.Fatalf("IdentityOf = %+v, want {%v %v}", identity, word, url)
	}
}
