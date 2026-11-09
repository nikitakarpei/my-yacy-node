package pairingrequests_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess/brokeraccesstest"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/pairingrequests"
)

const (
	secret    = "pairing-secret"
	replyWait = 5 * time.Second
)

type secretCheck struct{}

func (secretCheck) Check(client natsserver.ClientAuthentication) bool {
	options := client.GetOpts()
	return options.Username == brokeraccesstest.PeerUser || options.Password == secret ||
		len(client.GetTLSConnectionState().PeerCertificates) > 0
}

type pairingFixture struct {
	broker   brokeraccesstest.Broker
	requests chan *nats.Msg
}

func startPeer(
	t *testing.T,
	reply func(request companionprotocol.PairingRequest) any,
) pairingFixture {
	t.Helper()
	broker := brokeraccesstest.Start(t, secretCheck{})
	requests := make(chan *nats.Msg, 1)
	responder := broker.Connect(t)
	if _, err := responder.Subscribe(companionprotocol.PairingRequestSubject, func(msg *nats.Msg) {
		requests <- msg
		var request companionprotocol.PairingRequest
		_ = json.Unmarshal(msg.Data, &request)
		data, _ := json.Marshal(reply(request))
		_ = msg.Respond(data)
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := responder.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	return pairingFixture{broker: broker, requests: requests}
}

func TestLinkKindOfTellsPairLinksFromOtherText(t *testing.T) {
	for text, want := range map[string]string{
		companionprotocol.PairLink{Addresses: []string{"a:1"}, BrokerKeyHash: "h", Secret: "s"}.String(): pairingrequests.LinkPair,
		"yacypeer-companion://connect?nats=nats://peer":                                                  pairingrequests.LinkUnknownKind,
		"yacypeer-companion://pair?address=a:1&brokerKeyHash=h":                                          pairingrequests.LinkUnreadable,
		"https://example.org/": pairingrequests.LinkForeign,
	} {
		if got := pairingrequests.LinkKindOf(text); got != want {
			t.Errorf("kind of %q = %q, want %q", text, got, want)
		}
	}
}

func TestTheChallengeOfALinkBindsItsSecretAndBrokerKey(t *testing.T) {
	link := companionprotocol.PairLink{Addresses: []string{"a:1"}, BrokerKeyHash: "h", Secret: "s"}

	if !bytes.Equal(
		pairingrequests.ChallengeFrom(link.String()),
		companionprotocol.ChallengeFrom("s", "h"),
	) {
		t.Error("the challenge does not bind the secret and the broker key")
	}
	if pairingrequests.ChallengeFrom("https://example.org/") != nil {
		t.Error("text that is not a pair link has a challenge")
	}
}

func TestAnAcceptedRequestProvesTheKeyOfTheDevice(t *testing.T) {
	key := brokeraccesstest.RandomDeviceKey(t)
	peer := startPeer(t, func(companionprotocol.PairingRequest) any {
		return companionprotocol.AcceptedPairingReply(key.ID())
	})
	request := pairingrequests.NewRequest(peer.broker.LinkWith(secret), "Pixel 8", "1.2.0")
	request.AddCertificate([]byte("leaf"))

	if result := request.Send(key); result != pairingrequests.ResultAccepted {
		t.Fatalf("result = %q, want accepted", result)
	}
	sent := <-peer.requests
	var got companionprotocol.PairingRequest
	if err := json.Unmarshal(sent.Data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	publicKey, _ := x509.ParsePKIXPublicKey(key.PublicKeyInfo())
	challenge := companionprotocol.ChallengeFrom(secret, peer.broker.KeyHash)
	if !got.Proof.HoldsFor(publicKey.(*ecdsa.PublicKey), challenge) {
		t.Error("the request proves no key")
	}
	if got.ProtocolVersion != companionprotocol.ProtocolVersion || got.AppVersion != "1.2.0" ||
		len(got.Attestation) != 1 {
		t.Errorf("request = %+v, want protocol, app version and attestation", got)
	}
	if companionprotocol.SecretIDIn(sent.Reply) != companionprotocol.SecretIDFrom(secret) {
		t.Errorf("reply subject = %q, want the pairing inbox of the secret", sent.Reply)
	}
}

func TestARequestKeepsACertificateThatTheCallerReuses(t *testing.T) {
	sent := make(chan [][]byte, 1)
	peer := startPeer(t, func(request companionprotocol.PairingRequest) any {
		sent <- request.Attestation
		return companionprotocol.PairingReply{Outcome: companionprotocol.OutcomeAccepted}
	})
	request := pairingrequests.NewRequest(peer.broker.LinkWith(secret), "Pixel 8", "1.2.0")
	certificate := []byte("leaf")
	request.AddCertificate(certificate)
	copy(certificate, "used")

	request.Send(brokeraccesstest.RandomDeviceKey(t))

	if attestation := <-sent; len(attestation) != 1 || string(attestation[0]) != "leaf" {
		t.Errorf("attestation = %q, want the certificate as it was added", attestation)
	}
}

func TestEachRefusalReasonHasItsResult(t *testing.T) {
	for reason, want := range map[companionprotocol.RefusalReason]string{
		companionprotocol.ReasonAppOutdated:      pairingrequests.ResultAppOutdated,
		companionprotocol.ReasonPeerOutdated:     pairingrequests.ResultPeerOutdated,
		companionprotocol.ReasonAppNotOfficial:   pairingrequests.ResultAppNotOfficial,
		companionprotocol.ReasonDeviceNotTrusted: pairingrequests.ResultDeviceNotTrusted,
		companionprotocol.ReasonRequestInvalid:   pairingrequests.ResultRefused,
		"a reason of a later peer":               pairingrequests.ResultRefused,
	} {
		peer := startPeer(t, func(companionprotocol.PairingRequest) any {
			return companionprotocol.PairingReply{
				Outcome: companionprotocol.OutcomeRefused, Reason: reason,
			}
		})

		result := pairingrequests.NewRequest(peer.broker.LinkWith(secret), "Pixel 8", "1.2.0").
			Send(brokeraccesstest.RandomDeviceKey(t))

		if result != want {
			t.Errorf("result for %s = %q, want %q", reason, result, want)
		}
	}
}

func TestARequestWithoutAPairLinkIsNotSent(t *testing.T) {
	result := pairingrequests.NewRequest("https://example.org/", "Pixel 8", "1.2.0").
		Send(brokeraccesstest.RandomDeviceKey(t))

	if result != pairingrequests.ResultNotSent {
		t.Errorf("result = %q, want not sent", result)
	}
}

func TestAReplyThatIsNoPairingReplyIsUnreadable(t *testing.T) {
	peer := startPeer(t, func(companionprotocol.PairingRequest) any { return "not a reply" })

	result := pairingrequests.NewRequest(peer.broker.LinkWith(secret), "Pixel 8", "1.2.0").
		Send(brokeraccesstest.RandomDeviceKey(t))

	if result != pairingrequests.ResultReplyUnreadable {
		t.Errorf("result = %q, want the reply unreadable", result)
	}
}

func TestAnUnknownSecretHasExpired(t *testing.T) {
	peer := startPeer(t, func(companionprotocol.PairingRequest) any { return nil })

	result := pairingrequests.NewRequest(peer.broker.LinkWith("another secret"), "Pixel 8", "1.2.0").
		Send(brokeraccesstest.RandomDeviceKey(t))

	if result != pairingrequests.ResultSecretExpired {
		t.Errorf("result = %q, want the secret expired", result)
	}
}

func TestABrokerWithAnotherKeyGivesNoAnswer(t *testing.T) {
	peer := startPeer(t, func(companionprotocol.PairingRequest) any { return nil })
	link := companionprotocol.PairLink{
		Addresses: []string{peer.broker.Address()}, BrokerKeyHash: "another key", Secret: secret,
	}

	result := pairingrequests.NewRequest(link.String(), "Pixel 8", "1.2.0").
		Send(brokeraccesstest.RandomDeviceKey(t))

	if result != pairingrequests.ResultPeerUnreachable {
		t.Errorf("result = %q, want no answer", result)
	}
}

func TestAPairedDeviceAsksThePeerToUnpairIt(t *testing.T) {
	broker := brokeraccesstest.Start(t, secretCheck{})
	key := brokeraccesstest.RandomDeviceKey(t)
	unpairingRequests := make(chan *nats.Msg, 1)
	responder := broker.Connect(t)
	if _, err := responder.Subscribe(
		companionprotocol.UnpairingRequestSubjectOf(key.ID()),
		func(msg *nats.Msg) {
			unpairingRequests <- msg
			_ = msg.Respond(nil)
		},
	); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := responder.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	pairingrequests.Unpair(broker.LinkWith(secret), key)

	select {
	case <-unpairingRequests:
	case <-time.After(replyWait):
		t.Error("the peer got no unpairing request")
	}
}
