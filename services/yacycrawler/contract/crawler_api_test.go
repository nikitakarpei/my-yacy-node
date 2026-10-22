package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestAnAPIPathForAnIDCarriesTheIDEscaped(t *testing.T) {
	path := yacycrawlcontract.CrawlHostsPath.For("o/1 x")

	if path != "/crawls/o%2F1%20x/hosts" {
		t.Fatalf("CrawlHostsPath.For = %q", path)
	}
}
