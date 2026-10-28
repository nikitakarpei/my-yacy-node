// Package yacycrawlcontracttest builds the contract values of a test.
package yacycrawlcontracttest

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func RecrawlGraceOf(t *testing.T, length time.Duration) yacycrawlcontract.RecrawlGrace {
	t.Helper()
	recrawlGrace, err := yacycrawlcontract.RecrawlGraceOf(length)
	if err != nil {
		t.Fatalf("recrawl grace of %s: %v", length, err)
	}
	return recrawlGrace
}
