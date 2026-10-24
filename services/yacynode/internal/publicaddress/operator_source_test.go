package publicaddress_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/publicaddress"
)

func TestOperatorSourceGivesTheStatedHost(t *testing.T) {
	stated := hostsOf(t, "peer.example")[0]

	host, present := publicaddress.Stated(stated).Host(context.Background()).Get()

	if !present || host != stated {
		t.Fatalf("Host = %v (present %t), want %v", host, present, stated)
	}
}
