package peerwire_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

func TestIdentifiedCalleeHasTheHashOfItsIdentity(t *testing.T) {
	callee := peerwire.IdentifiedCallee(peerkeytest.Key(0).Identity())

	if callee.Hash() != peerkeytest.Key(0).Hash() {
		t.Errorf("Hash = %v, want %v", callee.Hash(), peerkeytest.Key(0).Hash())
	}
	if identity, _ := callee.Identity().Get(); identity != peerkeytest.Key(0).Identity() {
		t.Errorf("Identity = %v, want %v", identity, peerkeytest.Key(0).Identity())
	}
}

func TestUnidentifiedCalleeHasOnlyItsHash(t *testing.T) {
	callee := peerwire.UnidentifiedCallee(peerkeytest.Key(1).Hash())

	if callee.Hash() != peerkeytest.Key(1).Hash() ||
		callee.Identity() != yacymodel.None[yacymodel.PeerIdentity]() {
		t.Errorf("callee = %v, want hash %v without identity", callee, peerkeytest.Key(1).Hash())
	}
}
