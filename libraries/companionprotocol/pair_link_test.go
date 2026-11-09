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

func TestPairLinkFromRefusesTextOfAnotherSchemeAsNoCompanionLink(t *testing.T) {
	_, err := companionprotocol.PairLinkFrom("https://example.org/")

	if err == nil || errors.Is(err, companionprotocol.ErrUnreadableLink) ||
		errors.Is(err, companionprotocol.ErrUnknownLinkKind) {
		t.Errorf("error = %v, want one that names no companion link", err)
	}
}

func TestPairLinkFromTellsALinkOfAnUnknownKind(t *testing.T) {
	_, err := companionprotocol.PairLinkFrom("yacypeer-companion://connect?nats=nats://peer:4222")

	if !errors.Is(err, companionprotocol.ErrUnknownLinkKind) {
		t.Errorf("error = %v, want ErrUnknownLinkKind", err)
	}
}

func TestPairLinkFromCallsAPairLinkWithoutAFieldUnreadable(t *testing.T) {
	for _, text := range []string{
		"yacypeer-companion://pair?address=192.168.1.20:4223&brokerKeyHash=h",
		"yacypeer-companion://pair?address=&brokerKeyHash=h&secret=s",
	} {
		_, err := companionprotocol.PairLinkFrom(text)

		if !errors.Is(err, companionprotocol.ErrUnreadableLink) {
			t.Errorf("error for %q = %v, want ErrUnreadableLink", text, err)
		}
	}
}
