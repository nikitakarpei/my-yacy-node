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
	identity := postingidentity.Identity{Word: word, URL: url}
	keyOf := postingidentity.KeyLayout.Encode

	sameIdentity := postingidentity.Identity{Word: word, URL: url}
	if !bytes.Equal(keyOf(identity).Bytes(), keyOf(sameIdentity).Bytes()) {
		t.Fatal("one posting addresses two rows")
	}
	for name, other := range map[string]postingidentity.Identity{
		"another word": {Word: yacymodel.WordHash("w2"), URL: url},
		"another url":  {Word: word, URL: urlHash("u2")},
	} {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(keyOf(identity).Bytes(), keyOf(other).Bytes()) {
				t.Fatal("two postings share one row")
			}
		})
	}
}

func TestKeyRoundTripsToTheSamePosting(t *testing.T) {
	identity := postingidentity.Identity{Word: yacymodel.WordHash("w1"), URL: urlHash("u1")}

	encoded := postingidentity.KeyLayout.Encode(identity).Bytes()
	decoded, err := postingidentity.KeyLayout.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded != identity {
		t.Fatalf("Decode = %+v, want %+v", decoded, identity)
	}
}

func TestIdentityOfCarriesTheWordAndURLOfThePosting(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	identity := postingidentity.IdentityOf(yacymodel.RWIPosting{WordHash: word, URLHash: url})

	if identity.Word != word || identity.URL != url {
		t.Fatalf("IdentityOf = %+v, want {%v %v}", identity, word, url)
	}
}
