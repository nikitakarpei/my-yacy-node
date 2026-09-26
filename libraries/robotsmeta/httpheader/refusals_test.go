package httpheader_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
)

func TestEachValueAddsItsRefusals(t *testing.T) {
	refusals := httpheader.RefusalsOf([]string{"noindex", "nofollow"})

	if refusals != (robotsmeta.Refusals{RefusesIndexing: true, RefusesLinkDiscovery: true}) {
		t.Fatalf("refusals = %+v, want both", refusals)
	}
}

func TestAValueForANamedBotStatesNoRefusal(t *testing.T) {
	refusals := httpheader.RefusalsOf([]string{"googlebot: noindex, nofollow"})

	if refusals != (robotsmeta.Refusals{}) {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestADirectiveWithAValueNamesNoBot(t *testing.T) {
	for _, robotsTagValue := range []string{
		"unavailable_after: 25 Jun 2010 15:00:00 PST, noindex",
		"max-snippet: 20, noindex",
		"nosnippet, noindex, max-image-preview: large",
	} {
		refusals := httpheader.RefusalsOf([]string{robotsTagValue})

		if !refusals.RefusesIndexing {
			t.Errorf("%q yields %+v, want indexing refused", robotsTagValue, refusals)
		}
	}
}
