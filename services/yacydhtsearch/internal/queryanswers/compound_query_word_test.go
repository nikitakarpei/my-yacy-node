package queryanswers_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestEachPairOfAdjacentTermsSpellsOneCompoundQueryWord(t *testing.T) {
	t.Parallel()

	want := []queryanswers.CompoundQueryWord{
		{
			Word:       yacymodel.WordHash("archwiki"),
			FirstWord:  yacymodel.WordHash("arch"),
			SecondWord: yacymodel.WordHash("wiki"),
		},
		{
			Word:       yacymodel.WordHash("wikiinstall"),
			FirstWord:  yacymodel.WordHash("wiki"),
			SecondWord: yacymodel.WordHash("install"),
		},
	}
	got := queryanswers.CompoundQueryWordsFrom([]string{"arch", "wiki", "install"})

	if !slices.Equal(got, want) {
		t.Fatalf("the compound query words read %v, want %v", got, want)
	}
}

func TestAQueryOfOneTermSpellsNoCompoundQueryWord(t *testing.T) {
	t.Parallel()

	if got := queryanswers.CompoundQueryWordsFrom([]string{"arch"}); len(got) != 0 {
		t.Fatalf("the compound query words read %v, want none", got)
	}
}
