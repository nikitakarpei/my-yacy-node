package companionrequests_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

const (
	brokerKeyHash = "broker-key-hash"
	replyWait     = 5 * time.Second
)

var pairedAt = time.Date(2026, 11, 7, 14, 2, 11, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return pairedAt }

type begunOffers struct {
	pairings map[companionprotocol.SecretID]pairingoffers.OngoingPairing
	accepted map[companionprotocol.SecretID]companionprotocol.DeviceID
	refused  map[companionprotocol.SecretID]companionprotocol.RefusalReason
	mu       sync.Mutex
}

func (o *begunOffers) HoldForVerdict(
	_ context.Context,
	secretID companionprotocol.SecretID,
) (pairingoffers.OngoingPairing, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	pairing, found := o.pairings[secretID]
	return pairing, found
}

func (o *begunOffers) Accept(
	_ context.Context,
	secretID companionprotocol.SecretID,
	device companionprotocol.DeviceID,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.accepted[secretID] = device
}

func (o *begunOffers) Refuse(
	_ context.Context,
	secretID companionprotocol.SecretID,
	reason companionprotocol.RefusalReason,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.refused[secretID] = reason
}

type addedDevices struct {
	added []paireddevices.Device
	mu    sync.Mutex
}

func (d *addedDevices) Add(_ context.Context, device paireddevices.Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.added = append(d.added, device)
}

type pairingFixture struct {
	connection *nats.Conn
	offers     *begunOffers
	devices    *addedDevices
}

func connectTo(t *testing.T, url string) *nats.Conn {
	t.Helper()
	connection, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(connection.Close)
	return connection
}

func serve(
	t *testing.T,
	server func(context.Context, *nats.Conn) error,
	url string,
	subjects int,
) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ready := connectTo(t, url)
	done := make(chan error, 1)
	go func() { done <- server(ctx, ready) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	deadline := time.Now().Add(replyWait)
	for ready.NumSubscriptions() < subjects {
		if time.Now().After(deadline) {
			t.Fatal("the requests are not served")
		}
		time.Sleep(time.Millisecond)
	}
	if err := ready.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
}

func startPairing(t *testing.T) pairingFixture {
	t.Helper()
	url := natstestserver.Start(t)
	offers := &begunOffers{
		pairings: map[companionprotocol.SecretID]pairingoffers.OngoingPairing{},
		accepted: map[companionprotocol.SecretID]companionprotocol.DeviceID{},
		refused:  map[companionprotocol.SecretID]companionprotocol.RefusalReason{},
	}
	devices := &addedDevices{}
	requests := companionrequests.NewPairingAnswerer(offers, devices, fixedClock{}, brokerKeyHash)
	serve(t, requests.Serve, url, 1)
	return pairingFixture{connection: connectTo(t, url), offers: offers, devices: devices}
}

type attestedDevice struct {
	key         *ecdsa.PrivateKey
	attestation [][]byte
	id          companionprotocol.DeviceID
}

func newAttestedDevice(t *testing.T) attestedDevice {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    pairedAt,
		NotAfter:     pairedAt.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	return attestedDevice{
		key:         key,
		attestation: [][]byte{der},
		id:          companionprotocol.DeviceIDFrom(leaf.RawSubjectPublicKeyInfo),
	}
}

func (f pairingFixture) begin(secret companionprotocol.Secret) {
	f.offers.mu.Lock()
	defer f.offers.mu.Unlock()
	f.offers.pairings[companionprotocol.SecretIDFrom(secret)] = pairingoffers.OngoingPairing{
		Secret: secret, DeviceName: "Pixel 8",
	}
}

func (o *begunOffers) acceptedAs(secret companionprotocol.Secret) companionprotocol.DeviceID {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.accepted[companionprotocol.SecretIDFrom(secret)]
}

func (o *begunOffers) refusedFor(secret companionprotocol.Secret) companionprotocol.RefusalReason {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.refused[companionprotocol.SecretIDFrom(secret)]
}

func (d *addedDevices) all() []paireddevices.Device {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]paireddevices.Device{}, d.added...)
}

func (f pairingFixture) ask(
	t *testing.T,
	secret companionprotocol.Secret,
	request companionprotocol.PairingRequest,
) companionprotocol.PairingReply {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	inbox := companionprotocol.PairingInboxPrefixFor(secret) + "." + nats.NewInbox()
	subscription, err := f.connection.SubscribeSync(inbox)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	if err := f.connection.PublishRequest(
		companionprotocol.PairingRequestSubject, inbox, data,
	); err != nil {
		t.Fatalf("PublishRequest: %v", err)
	}
	message, err := subscription.NextMsg(replyWait)
	if err != nil {
		t.Fatalf("no reply: %v", err)
	}
	var reply companionprotocol.PairingReply
	if err := json.Unmarshal(message.Data, &reply); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return reply
}

func requestOf(
	t *testing.T,
	device attestedDevice,
	secret companionprotocol.Secret,
) companionprotocol.PairingRequest {
	t.Helper()
	proof, err := companionprotocol.ProofOf(
		device.key, companionprotocol.ChallengeFrom(secret, brokerKeyHash),
	)
	if err != nil {
		t.Fatalf("ProofOf: %v", err)
	}
	return companionprotocol.PairingRequest{
		ProtocolVersion: companionprotocol.ProtocolVersion,
		AppVersion:      "1.2.0",
		Attestation:     device.attestation,
		Proof:           proof,
	}
}

func TestADeviceThatProvesItsKeyPairs(t *testing.T) {
	fixture := startPairing(t)
	device := newAttestedDevice(t)
	secret := companionprotocol.RandomSecret()
	fixture.begin(secret)

	reply := fixture.ask(t, secret, requestOf(t, device, secret))

	if reply.Outcome != companionprotocol.OutcomeAccepted || reply.DeviceID != device.id {
		t.Fatalf("reply = %+v, want accepted as %s", reply, device.id)
	}
	want := paireddevices.Device{
		ID: device.id, Name: "Pixel 8", PairedAt: pairedAt, AppVersion: "1.2.0",
	}
	if added := fixture.devices.all(); len(added) != 1 || added[0] != want {
		t.Errorf("added devices = %+v, want %+v", added, want)
	}
	if accepted := fixture.offers.acceptedAs(secret); accepted != device.id {
		t.Errorf("accepted as %q, want %s", accepted, device.id)
	}
}

func TestAPairingRequestIsRefusedForEachFlaw(t *testing.T) {
	device := newAttestedDevice(t)
	for _, flaw := range []struct {
		name   string
		change func(request *companionprotocol.PairingRequest)
		want   companionprotocol.RefusalReason
	}{
		{
			"older protocol", func(r *companionprotocol.PairingRequest) { r.ProtocolVersion = 0 },
			companionprotocol.ReasonAppOutdated,
		},
		{
			"newer protocol", func(r *companionprotocol.PairingRequest) { r.ProtocolVersion = 2 },
			companionprotocol.ReasonPeerOutdated,
		},
		{"proof of another challenge", func(r *companionprotocol.PairingRequest) {
			r.Proof = requestOf(t, device, "another secret").Proof
		}, companionprotocol.ReasonRequestInvalid},
		{
			"no attestation", func(r *companionprotocol.PairingRequest) { r.Attestation = nil },
			companionprotocol.ReasonRequestInvalid,
		},
		{"unreadable attestation", func(r *companionprotocol.PairingRequest) {
			r.Attestation = [][]byte{[]byte("not a certificate")}
		}, companionprotocol.ReasonRequestInvalid},
	} {
		t.Run(flaw.name, func(t *testing.T) {
			fixture := startPairing(t)
			secret := companionprotocol.RandomSecret()
			fixture.begin(secret)
			request := requestOf(t, device, secret)
			flaw.change(&request)

			reply := fixture.ask(t, secret, request)

			if reply.Outcome != companionprotocol.OutcomeRefused || reply.Reason != flaw.want {
				t.Errorf("reply = %+v, want refused for %s", reply, flaw.want)
			}
			if refused := fixture.offers.refusedFor(secret); refused != flaw.want {
				t.Errorf("refused for %q, want %s", refused, flaw.want)
			}
			if added := fixture.devices.all(); len(added) != 0 {
				t.Errorf("added devices = %+v, want none", added)
			}
		})
	}
}

func TestAnUnreadablePairingRequestIsRefused(t *testing.T) {
	fixture := startPairing(t)
	secret := companionprotocol.RandomSecret()
	fixture.begin(secret)
	inbox := companionprotocol.PairingInboxPrefixFor(secret) + ".reply"
	subscription, err := fixture.connection.SubscribeSync(inbox)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}

	if err := fixture.connection.PublishRequest(
		companionprotocol.PairingRequestSubject, inbox, []byte("{"),
	); err != nil {
		t.Fatalf("PublishRequest: %v", err)
	}

	message, err := subscription.NextMsg(replyWait)
	if err != nil {
		t.Fatalf("no reply: %v", err)
	}
	var reply companionprotocol.PairingReply
	if err := json.Unmarshal(message.Data, &reply); err != nil ||
		reply.Reason != companionprotocol.ReasonRequestInvalid {
		t.Errorf("reply = %s, want refused as invalid", message.Data)
	}
}

func TestAPairingRequestWithoutABegunPairingGetsNoReply(t *testing.T) {
	fixture := startPairing(t)
	secret := companionprotocol.RandomSecret()
	device := newAttestedDevice(t)
	inbox := companionprotocol.PairingInboxPrefixFor(secret) + ".reply"
	subscription, err := fixture.connection.SubscribeSync(inbox)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	data, _ := json.Marshal(requestOf(t, device, secret))

	if err := fixture.connection.PublishRequest(
		companionprotocol.PairingRequestSubject, inbox, data,
	); err != nil {
		t.Fatalf("PublishRequest: %v", err)
	}

	begunSecret := companionprotocol.RandomSecret()
	fixture.begin(begunSecret)
	fixture.ask(t, begunSecret, requestOf(t, device, begunSecret))

	if pending, _, err := subscription.Pending(); err != nil || pending != 0 {
		t.Errorf("replies = %d (%v), want none", pending, err)
	}
}
