package nodeidentity_test

import (
	"regexp"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

var readablePeerName = regexp.MustCompile(`^[a-z]{3,}-[a-z]{3,}-[0-9]{4}$`)

func TestPeerNameFromHashReadsAsWordsAndHoldsForOneHash(t *testing.T) {
	hash := hashOf(t, "0123456789AB")

	derived, err := nodeidentity.PeerNameFromHash(hash)
	if err != nil {
		t.Fatalf("derive peer name: %v", err)
	}
	if !readablePeerName.MatchString(derived.String()) {
		t.Fatalf("PeerNameFromHash = %q, want two words and a number", derived)
	}
	if _, err := yacymodel.ParsePeerName(derived.String()); err != nil {
		t.Fatalf("ParsePeerName(%q) = %v, want a name the network accepts", derived, err)
	}

	again, err := nodeidentity.PeerNameFromHash(hash)
	if err != nil {
		t.Fatalf("derive peer name again: %v", err)
	}
	if again != derived {
		t.Fatalf("PeerNameFromHash = %q on a second call, want %q", again, derived)
	}
}

func TestPeerNameFromHashSeparatesHashes(t *testing.T) {
	derivedFrom := map[string]string{}
	for _, text := range []string{
		"0123456789AB", "BA9876543210", "aaaaaaaaaaaa", "aaaaaaaaaaab", "zzzzzzzzzzzz",
	} {
		derived, err := nodeidentity.PeerNameFromHash(hashOf(t, text))
		if err != nil {
			t.Fatalf("derive peer name from %q: %v", text, err)
		}

		if other, clashing := derivedFrom[derived.String()]; clashing {
			t.Fatalf("hashes %q and %q both derive %q", other, text, derived)
		}
		derivedFrom[derived.String()] = text
	}
}

func hashOf(t *testing.T, text string) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(text)
	if err != nil {
		t.Fatalf("parse hash %q: %v", text, err)
	}

	return hash
}
