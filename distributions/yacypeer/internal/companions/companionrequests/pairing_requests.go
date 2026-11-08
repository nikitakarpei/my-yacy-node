// Package companionrequests answers the requests that companions send to the broker of the peer:
// the pairing request of a device that pairs, and the settings requests, app reports and unpair
// requests of paired devices.
package companionrequests

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

var (
	errOlderProtocol = errors.New("the app speaks an older protocol")
	errNewerProtocol = errors.New("the app speaks a newer protocol")
	errNoAttestation = errors.New("the request holds no attestation")
	errNotAnECKey    = errors.New("the attested key is not an EC key")
	errProofInvalid  = errors.New("the proof does not hold for the attested key")
)

type PairingOffers interface {
	HoldForVerdict(
		ctx context.Context,
		secretID companionprotocol.SecretID,
	) (pairingoffers.OngoingPairing, bool)
	Accept(
		ctx context.Context,
		secretID companionprotocol.SecretID,
		device companionprotocol.DeviceID,
	)
	Refuse(
		ctx context.Context,
		secretID companionprotocol.SecretID,
		reason companionprotocol.Reason,
	)
}

type NewDevices interface {
	Add(ctx context.Context, device paireddevices.Device)
}

type Clock interface {
	Now() time.Time
}

type PairingRequests struct {
	offers        PairingOffers
	devices       NewDevices
	clock         Clock
	brokerKeyHash string
}

func NewPairingRequests(
	offers PairingOffers,
	devices NewDevices,
	clock Clock,
	brokerKeyHash string,
) *PairingRequests {
	return &PairingRequests{
		offers:        offers,
		devices:       devices,
		clock:         clock,
		brokerKeyHash: brokerKeyHash,
	}
}

func (p *PairingRequests) Serve(ctx context.Context, connection *nats.Conn) error {
	return serve(ctx, connection, map[string]nats.MsgHandler{
		companionprotocol.PairingRequestSubject: func(request *nats.Msg) {
			p.answer(ctx, request)
		},
	})
}

func (p *PairingRequests) answer(ctx context.Context, request *nats.Msg) {
	secretID := companionprotocol.SecretIDFromReply(request.Reply)
	pairing, held := p.offers.HoldForVerdict(ctx, secretID)
	if !held {
		slog.WarnContext(ctx, "pairing request left unanswered",
			slog.String("reply", request.Reply),
		)
		return
	}
	device, reason := p.verdictOn(ctx, pairing, request.Data)
	if reason != "" {
		p.offers.Refuse(ctx, secretID, reason)
		respond(ctx, request, companionprotocol.PairingReply{
			Outcome: companionprotocol.OutcomeRefused, Reason: reason,
		})
		return
	}
	p.devices.Add(ctx, device)
	p.offers.Accept(ctx, secretID, device.ID)
	respond(ctx, request, companionprotocol.PairingReply{
		Outcome: companionprotocol.OutcomeAccepted, Device: device.ID,
	})
}

func (p *PairingRequests) verdictOn(
	ctx context.Context,
	pairing pairingoffers.OngoingPairing,
	data []byte,
) (paireddevices.Device, companionprotocol.Reason) {
	device, reason, cause := p.checked(pairing, data)
	if reason != "" {
		slog.WarnContext(ctx, "pairing request refused",
			slog.String("deviceName", string(pairing.DeviceName)),
			slog.String("reason", string(reason)),
			slog.Any("cause", cause),
		)
	}
	return device, reason
}

func (p *PairingRequests) checked(
	pairing pairingoffers.OngoingPairing,
	data []byte,
) (paireddevices.Device, companionprotocol.Reason, error) {
	var request companionprotocol.PairingRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return paireddevices.Device{}, companionprotocol.ReasonRequestInvalid,
			fmt.Errorf("read the pairing request: %w", err)
	}
	switch {
	case request.Protocol < companionprotocol.ProtocolVersion:
		return paireddevices.Device{}, companionprotocol.ReasonAppOutdated, errOlderProtocol
	case request.Protocol > companionprotocol.ProtocolVersion:
		return paireddevices.Device{}, companionprotocol.ReasonPeerOutdated, errNewerProtocol
	}
	attested, err := attestedKeyOf(request.Attestation)
	if err != nil {
		return paireddevices.Device{}, companionprotocol.ReasonRequestInvalid, err
	}
	challenge := companionprotocol.ChallengeFor(pairing.Secret, p.brokerKeyHash)
	if !companionprotocol.IsProofOf(attested.publicKey, challenge, request.Proof) {
		return paireddevices.Device{}, companionprotocol.ReasonRequestInvalid, errProofInvalid
	}
	return paireddevices.Device{
		ID:         companionprotocol.DeviceIDOf(attested.keyInfo),
		Name:       pairing.DeviceName,
		PairedAt:   p.clock.Now(),
		AppVersion: request.AppVersion,
	}, "", nil
}

type attestedKey struct {
	publicKey *ecdsa.PublicKey
	keyInfo   []byte
}

func attestedKeyOf(attestation [][]byte) (attestedKey, error) {
	if len(attestation) == 0 {
		return attestedKey{}, errNoAttestation
	}
	leaf, err := x509.ParseCertificate(attestation[0])
	if err != nil {
		return attestedKey{}, fmt.Errorf("read the attested key: %w", err)
	}
	publicKey, isECDSA := leaf.PublicKey.(*ecdsa.PublicKey)
	if !isECDSA {
		return attestedKey{}, errNotAnECKey
	}
	return attestedKey{publicKey: publicKey, keyInfo: leaf.RawSubjectPublicKeyInfo}, nil
}

func respond(ctx context.Context, request *nats.Msg, reply any) {
	data, err := json.Marshal(reply)
	if err == nil {
		err = request.Respond(data)
	}
	if err != nil {
		slog.WarnContext(ctx, "companion reply not sent",
			slog.String("subject", request.Subject), slog.Any("error", err),
		)
	}
}

func serve(
	ctx context.Context,
	connection *nats.Conn,
	handlers map[string]nats.MsgHandler,
) error {
	subscriptions := make([]*nats.Subscription, 0, len(handlers))
	for subject, handler := range handlers {
		subscription, err := connection.Subscribe(subject, handler)
		if err != nil {
			return fmt.Errorf("subscribe to %q: %w", subject, err)
		}
		subscriptions = append(subscriptions, subscription)
	}
	<-ctx.Done()
	for _, subscription := range subscriptions {
		_ = subscription.Unsubscribe()
	}
	return nil
}
