package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func mustHash(t *testing.T, raw string) yacymodel.Hash {
	t.Helper()
	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func mustURLHash(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatal(err)
	}

	return hash
}
