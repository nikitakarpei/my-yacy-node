package companionprotocol_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestAPairLinkReadsBackAsTheSameLink(t *testing.T) {
	link := companionprotocol.PairLink{
		Addresses:     []string{"192.168.1.20:4223", "100.101.102.103:4223", "[2001:db8::7]:4223"},
		BrokerKeyHash: "key-hash",
		Secret:        companionprotocol.RandomSecret(),
	}

	read, err := companionprotocol.PairLinkFrom(link.String())
	if err != nil {
		t.Fatalf("PairLinkFrom: %v", err)
	}

	if !reflect.DeepEqual(read, link) {
		t.Errorf("read link = %+v, want %+v", read, link)
	}
}

func TestPairLinkFromRefusesTextOfAnotherScheme(t *testing.T) {
	_, err := companionprotocol.PairLinkFrom("https://example.org/")

	if !errors.Is(err, companionprotocol.ErrNotACompanionLink) {
		t.Errorf("error = %v, want ErrNotACompanionLink", err)
	}
}

func TestPairLinkFromCallsALinkOfAnUnknownKindUnreadable(t *testing.T) {
	for _, text := range []string{
		companionprotocol.Scheme + "://connect?nats=nats://peer:4222",
		companionprotocol.Scheme + "://pair?address=192.168.1.20:4223&brokerKeyHash=h",
		companionprotocol.Scheme + "://pair?address=&brokerKeyHash=h&secret=s",
	} {
		_, err := companionprotocol.PairLinkFrom(text)

		if !errors.Is(err, companionprotocol.ErrUnreadableLink) {
			t.Errorf("error for %q = %v, want ErrUnreadableLink", text, err)
		}
	}
}
