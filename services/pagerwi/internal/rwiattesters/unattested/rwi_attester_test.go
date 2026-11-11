package unattested_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/rwiattesters/unattested"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestEveryRWIGoesWithoutAttestationAndIsDecided(t *testing.T) {
	bundle, decided := unattested.RWIAttester{}.Attest(
		context.Background(),
		pagescrapecontract.PageOfferMessage{},
		pagescrapecontract.OfferedPage{},
		pagerwicontract.PageRWI{},
	)

	if !decided || bundle != nil {
		t.Errorf("bundle = %s, decided = %v", bundle, decided)
	}
}
