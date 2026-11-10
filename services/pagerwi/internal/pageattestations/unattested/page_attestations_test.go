package unattested_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageattestations/unattested"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestEveryRWIGoesWithoutAttestationAndIsReady(t *testing.T) {
	attestation, ready := unattested.PageAttestations{}.AttestationOf(
		context.Background(),
		pagescrapecontract.PageOfferMessage{},
		pagescrapecontract.OfferedPage{},
		pagerwicontract.PageRWI{},
	)

	if !ready || attestation != nil {
		t.Errorf("attestation = %s, ready = %v", attestation, ready)
	}
}
