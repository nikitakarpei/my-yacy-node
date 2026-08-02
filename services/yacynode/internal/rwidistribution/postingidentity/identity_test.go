package postingidentity

import (
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

func TestKeyMatchesWordAndURLBytes(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	got := string(IdentityOf(word, url).Key())
	want := word.String() + url.String()

	if got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}
}

func TestIdentityOfCarriesWordAndURL(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	identity := IdentityOf(word, url)

	if identity.Word != word || identity.URL != url {
		t.Fatalf("IdentityOf = %+v, want {%v %v}", identity, word, url)
	}
}
