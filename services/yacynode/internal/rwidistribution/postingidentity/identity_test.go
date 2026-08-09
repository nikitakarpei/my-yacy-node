package postingidentity

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
	posting := IdentityOf(word, url)

	if !bytes.Equal(posting.Key(), IdentityOf(word, url).Key()) {
		t.Fatal("one posting addresses two rows")
	}
	for name, other := range map[string]Identity{
		"another word": IdentityOf(yacymodel.WordHash("w2"), url),
		"another url":  IdentityOf(word, urlHash("u2")),
	} {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(posting.Key(), other.Key()) {
				t.Fatal("two postings share one row")
			}
		})
	}
}

func TestIdentityOfCarriesWordAndURL(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	identity := IdentityOf(word, url)

	if identity.Word != word || identity.URL != url {
		t.Fatalf("IdentityOf = %+v, want {%v %v}", identity, word, url)
	}
}
