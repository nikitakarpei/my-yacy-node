package robotsmeta_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
)

func TestEachRuleStatesItsRefusal(t *testing.T) {
	for _, testCase := range []struct {
		ruleList string
		want     robotsmeta.Refusals
	}{
		{ruleList: ""},
		{ruleList: "index, follow, nosnippet"},
		{ruleList: "noindex", want: robotsmeta.Refusals{RefusesIndexing: true}},
		{ruleList: "nofollow", want: robotsmeta.Refusals{RefusesLinkDiscovery: true}},
		{
			ruleList: "none",
			want:     robotsmeta.Refusals{RefusesIndexing: true, RefusesLinkDiscovery: true},
		},
		{
			ruleList: " NoIndex , NoFollow ",
			want:     robotsmeta.Refusals{RefusesIndexing: true, RefusesLinkDiscovery: true},
		},
	} {
		if got := robotsmeta.RefusalsIn(testCase.ruleList); got != testCase.want {
			t.Errorf("%q yields %+v, want %+v", testCase.ruleList, got, testCase.want)
		}
	}
}

func TestRefusalsWithOtherRefusalsHoldBoth(t *testing.T) {
	refusals := robotsmeta.Refusals{RefusesIndexing: true}.
		With(robotsmeta.Refusals{RefusesLinkDiscovery: true})

	if refusals != (robotsmeta.Refusals{RefusesIndexing: true, RefusesLinkDiscovery: true}) {
		t.Fatalf("refusals = %+v, want both", refusals)
	}
}
