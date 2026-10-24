package publicaddress_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/publicaddress"
)

func TestStatedAddressKeepsTheStatedHost(t *testing.T) {
	stated := hostsOf(t, "peer.example")[0]
	address := publicaddress.For(yacymodel.Some(stated))

	changed := address.Settle(context.Background(), hostsOf(t,
		"203.0.113.9", "203.0.113.9", "203.0.113.9",
	))

	if changed {
		t.Fatal("Settle = true, want the stated host kept")
	}
	if host, present := address.Host(context.Background()).Get(); !present || host != stated {
		t.Fatalf("Host = %v (present %t), want %v", host, present, stated)
	}
}
