package unattested_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/rwiattesters/unattested"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestEveryRWIGoesWithoutAttestationAndIsReady(t *testing.T) {
	attestation, issued := unattested.RWIAttester{}.Attest(
		context.Background(),
		pagescrapecontract.PageOfferMessage{},
		pagescrapecontract.OfferedPage{},
		pagerwicontract.PageRWI{},
	)

	if !issued || attestation != nil {
		t.Errorf("attestation = %s, issued = %v", attestation, issued)
	}
}
