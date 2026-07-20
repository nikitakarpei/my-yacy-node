package yacymodel

import (
	"errors"
	"testing"
)

func TestParsePeerName(t *testing.T) {
	name, err := ParsePeerName("peer-node_1")
	if err != nil || name.String() != "peer-node_1" {
		t.Fatalf("ParsePeerName = %q, %v", name.String(), err)
	}
}

func TestParsePeerNameRejects(t *testing.T) {
	for _, s := range []string{"ab", "has space", "tag,x", string(make([]byte, 81))} {
		if _, err := ParsePeerName(s); !errors.Is(err, ErrBadPeerName) {
			t.Fatalf("ParsePeerName(%q) = %v, want ErrBadPeerName", s, err)
		}
	}
}
