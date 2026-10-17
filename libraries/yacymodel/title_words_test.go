package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTitleWordsCountWordsBetweenRunsOfSpaces(t *testing.T) {
	for title, want := range map[string]int{
		"":              1,
		"Hello":         1,
		"Hello World":   2,
		"Hello   World": 2,
		"Hello · World": 3,
		" Hello World":  3,
		"Hello World ":  2,
		"   ":           0,
		"Hello\tWorld":  1,
	} {
		if got := yacymodel.TitleWordsOf(title); got != want {
			t.Errorf("TitleWordsOf(%q) = %d, want %d", title, got, want)
		}
	}
}
