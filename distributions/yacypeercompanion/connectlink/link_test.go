package connectlink_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/connectlink"
)

func TestLinkFromReadsTheBrokerAndThePartitionExponent(t *testing.T) {
	link, err := connectlink.LinkFrom(
		"yacypeer-companion://connect?nats=nats%3A%2F%2F192.168.1.20%3A4222&partition-exponent=4",
	)
	if err != nil {
		t.Fatalf("LinkFrom: %v", err)
	}
	if link.NATSURL != "nats://192.168.1.20:4222" || link.PartitionExponent != 4 {
		t.Errorf("link = %+v", link)
	}
}

func TestLinkFromRefusesTextThatIsNotAConnectLink(t *testing.T) {
	for _, text := range []string{
		"https://example.org/",
		"yacypeer-companion://elsewhere?nats=nats://h:4222&partition-exponent=4",
		"yacypeer-companion://connect?nats=http://h:4222&partition-exponent=4",
		"yacypeer-companion://connect?partition-exponent=4",
		"yacypeer-companion://connect?nats=nats://h:4222",
		"yacypeer-companion://connect?nats=nats://h:4222&partition-exponent=63",
		"yacypeer-companion://connect?nats=nats://h:4222&partition-exponent=-1",
		"%zz",
	} {
		if _, err := connectlink.LinkFrom(text); err == nil {
			t.Errorf("LinkFrom(%q): want an error", text)
		}
	}
}
