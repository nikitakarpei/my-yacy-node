package pagerobots_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagemarkup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerobots"
)

func refusalsFrom(t *testing.T, markup string) pagerobots.Refusals {
	t.Helper()
	parsed, err := pagemarkup.MarkupFrom(t.Context(), "text/html", []byte(markup))
	if err != nil {
		t.Fatalf("MarkupFrom: %v", err)
	}
	return pagerobots.RefusalsFrom(parsed)
}

func TestAPageWithoutMetaRobotsRefusesNothing(t *testing.T) {
	refusals := refusalsFrom(t, `<html><body><a href="/next">next</a></body></html>`)

	if refusals.RefusesIndexing || refusals.RefusesLinkDiscovery {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestMetaRobotsStatesEachRefusal(t *testing.T) {
	for _, testCase := range []struct {
		content        string
		wantNoIndex    bool
		wantNoDiscover bool
	}{
		{content: "noindex", wantNoIndex: true},
		{content: "nofollow", wantNoDiscover: true},
		{content: "none", wantNoIndex: true, wantNoDiscover: true},
		{content: "NoIndex, NoFollow", wantNoIndex: true, wantNoDiscover: true},
		{content: "index, follow"},
	} {
		refusals := refusalsFrom(t,
			`<html><head><meta name="ROBOTS" content="`+testCase.content+`"></head></html>`,
		)

		if refusals.RefusesIndexing != testCase.wantNoIndex ||
			refusals.RefusesLinkDiscovery != testCase.wantNoDiscover {
			t.Errorf("content %q yields %+v", testCase.content, refusals)
		}
	}
}

func TestAMetaTagForSomeoneElseStatesNoRefusal(t *testing.T) {
	refusals := refusalsFrom(t,
		`<html><head><meta name="googlebot" content="noindex">`+
			`<meta content="noindex"><meta name="robots"></head></html>`,
	)

	if refusals.RefusesIndexing || refusals.RefusesLinkDiscovery {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}
