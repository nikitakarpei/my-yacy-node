package publicaddress_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/publicaddress"
)

func TestOperatorSourceKeepsTheStatedHostWhateverPeersReport(t *testing.T) {
	stated := hostsOf(t, "peer.example")[0]
	source := publicaddress.For(yacymodel.Some(stated))

	source.ObserveReportedHosts(context.Background(), hostsOf(t,
		"203.0.113.9", "203.0.113.9", "203.0.113.9",
	))

	if host, present := source.Host(context.Background()).Get(); !present || host != stated {
		t.Fatalf("Host = %v (present %t), want %v", host, present, stated)
	}
}
