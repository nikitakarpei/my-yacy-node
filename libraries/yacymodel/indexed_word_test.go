package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAWordOfTwoLettersIsIndexed(t *testing.T) {
	t.Parallel()

	if !yacymodel.WordIsIndexed("go") {
		t.Fatal("WordIsIndexed(go) = false, want true")
	}
}

func TestAWordOfOneLetterIsNotIndexed(t *testing.T) {
	t.Parallel()

	if yacymodel.WordIsIndexed("1") {
		t.Fatal("WordIsIndexed(1) = true, want false")
	}
}

func TestAWordOfOneMultibyteLetterIsNotIndexed(t *testing.T) {
	t.Parallel()

	if yacymodel.WordIsIndexed("ä") {
		t.Fatal("WordIsIndexed(ä) = true, want false")
	}
}
