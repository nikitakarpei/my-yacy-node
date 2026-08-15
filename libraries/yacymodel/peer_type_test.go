package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestParsePeerType(t *testing.T) {
	for _, valid := range []string{"virgin", "junior", "mentee", "senior", "mentor", "principal"} {
		pt, err := yacymodel.ParsePeerType(valid)
		if err != nil {
			t.Errorf("ParsePeerType(%q) = %v", valid, err)
		}
		if pt.String() != valid {
			t.Errorf("String() = %q, want %q", pt, valid)
		}
	}
	if _, err := yacymodel.ParsePeerType("master"); !errors.Is(err, yacymodel.ErrInvalidPeerType) {
		t.Fatalf("ParsePeerType invalid = %v, want ErrInvalidPeerType", err)
	}
}
