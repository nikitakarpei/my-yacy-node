package featurehashing_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel/featurehashing"
)

func TestOneFeatureGetsTheIndexScikitLearnGivesIt(t *testing.T) {
	indexOfFeature := map[string]int32{
		"a":           354738,
		"ab":          10401,
		"abc":         158726,
		"abcd":        878442,
		"hello world": 167695,
		"Ünïcode":     672714,
	}
	for feature, index := range indexOfFeature {
		entries := featurehashing.EntriesFrom([]string{feature})
		if want := []featurehashing.Entry{{Index: index, Value: 1}}; !slices.Equal(entries, want) {
			t.Errorf("EntriesFrom(%q) = %v, want %v", feature, entries, want)
		}
	}
}

func TestRepeatedFeaturesAreCountedAndScaledToUnitLength(t *testing.T) {
	entries := featurehashing.EntriesFrom([]string{"div", "p", "div", "div p", "p div"})

	want := []featurehashing.Entry{
		{Index: 8294, Value: 0.3779644730092272},
		{Index: 477268, Value: 0.7559289460184544},
		{Index: 665046, Value: 0.3779644730092272},
		{Index: 883823, Value: 0.3779644730092272},
	}
	if !slices.Equal(entries, want) {
		t.Errorf("EntriesFrom = %v, want %v", entries, want)
	}
}

func TestNoFeaturesGiveNoEntries(t *testing.T) {
	if entries := featurehashing.EntriesFrom(nil); len(entries) != 0 {
		t.Errorf("EntriesFrom(nil) = %v, want none", entries)
	}
}
