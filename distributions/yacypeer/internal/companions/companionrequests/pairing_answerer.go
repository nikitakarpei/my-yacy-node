// Package companionrequests answers the requests that companions send to the broker of the peer:
// the pairing request of a device that pairs, and the settings requests, app reports and unpairing
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
		deviceID companionprotocol.DeviceID,
	)
	Refuse(
		ctx context.Context,
		secretID companionprotocol.SecretID,
		reason companionprotocol.RefusalReason,
	)
}

type AddableDevices interface {
	Add(ctx context.Context, device paireddevices.Device)
}

type Clock interface {
	Now() time.Time
}

type PairingAnswerer struct {
	offers        PairingOffers
	devices       AddableDevices
	clock         Clock
	brokerKeyHash companionprotocol.BrokerKeyHash
}

func NewPairingAnswerer(
	offers PairingOffers,
	devices AddableDevices,
	clock Clock,
	brokerKeyHash companionprotocol.BrokerKeyHash,
) *PairingAnswerer {
	return &PairingAnswerer{
		offers:        offers,
		devices:       devices,
		clock:         clock,
		brokerKeyHash: brokerKeyHash,
	}
}

func (p *PairingAnswerer) Subscribe(
	ctx context.Context,
	connection *nats.Conn,
) (*Subscriptions, error) {
	return subscribeTo(connection, map[string]nats.MsgHandler{
		companionprotocol.PairingRequestSubject: func(message *nats.Msg) {
			p.answer(ctx, message)
		},
	})
}

func (p *PairingAnswerer) answer(ctx context.Context, message *nats.Msg) {
	pairing, held := p.holdForVerdict(ctx, message.Reply)
	if !held {
		return
	}
	request, refusal := p.provenRequestFrom(message.Data, pairing.Secret)
	if refusal != nil {
		p.refuse(ctx, message, pairing, refusal)
		return
	}
	p.accept(ctx, message, pairing, request)
}

func (p *PairingAnswerer) holdForVerdict(
	ctx context.Context,
	inboxSubject string,
) (pairingoffers.OngoingPairing, bool) {
	pairing, held := p.offers.HoldForVerdict(ctx, companionprotocol.SecretIDIn(inboxSubject))
	if !held {
		slog.WarnContext(ctx, "pairing request left unanswered",
			slog.String("replySubject", inboxSubject),
		)
	}
	return pairing, held
}

type provenRequest struct {
	appVersion string
	deviceID   companionprotocol.DeviceID
}

type pairingRefusal struct {
	reason companionprotocol.RefusalReason
	cause  error
}

func (p *PairingAnswerer) provenRequestFrom(
	data []byte,
	secret companionprotocol.Secret,
) (provenRequest, *pairingRefusal) {
	request, refusal := pairingRequestFrom(data)
	if refusal != nil {
		return provenRequest{}, refusal
	}
	if refusal := protocolRefusalOf(request.ProtocolVersion); refusal != nil {
		return provenRequest{}, refusal
	}
	key, refusal := p.provenKeyFrom(request, secret)
	if refusal != nil {
		return provenRequest{}, refusal
	}
	return provenRequest{
		appVersion: request.AppVersion,
		deviceID:   companionprotocol.DeviceIDFrom(key.publicKeyInfo),
	}, nil
}

func pairingRequestFrom(data []byte) (companionprotocol.PairingRequest, *pairingRefusal) {
	var request companionprotocol.PairingRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return request, &pairingRefusal{
			reason: companionprotocol.ReasonRequestInvalid,
			cause:  fmt.Errorf("read the pairing request: %w", err),
		}
	}
	return request, nil
}

func protocolRefusalOf(protocolVersion int) *pairingRefusal {
	switch {
	case protocolVersion < companionprotocol.ProtocolVersion:
		return &pairingRefusal{reason: companionprotocol.ReasonAppOutdated, cause: errOlderProtocol}
	case protocolVersion > companionprotocol.ProtocolVersion:
		return &pairingRefusal{
			reason: companionprotocol.ReasonPeerOutdated,
			cause:  errNewerProtocol,
		}
	}
	return nil
}

func (p *PairingAnswerer) provenKeyFrom(
	request companionprotocol.PairingRequest,
	secret companionprotocol.Secret,
) (attestedKey, *pairingRefusal) {
	key, err := attestedKeyFrom(request.Attestation)
	if err != nil {
		return attestedKey{}, &pairingRefusal{
			reason: companionprotocol.ReasonRequestInvalid,
			cause:  err,
		}
	}
	challenge := companionprotocol.ChallengeFrom(secret, p.brokerKeyHash)
	if !request.Proof.HoldsFor(key.publicKey, challenge) {
		return attestedKey{}, &pairingRefusal{
			reason: companionprotocol.ReasonRequestInvalid,
			cause:  errProofInvalid,
		}
	}
	return key, nil
}

type attestedKey struct {
	publicKey     *ecdsa.PublicKey
	publicKeyInfo []byte
}

func attestedKeyFrom(attestation [][]byte) (attestedKey, error) {
	if len(attestation) == 0 {
		return attestedKey{}, errNoAttestation
	}
	leaf, err := x509.ParseCertificate(attestation[0])
	if err != nil {
		return attestedKey{}, fmt.Errorf("read the attested key: %w", err)
	}
	publicKey, isECKey := leaf.PublicKey.(*ecdsa.PublicKey)
	if !isECKey {
		return attestedKey{}, errNotAnECKey
	}
	return attestedKey{publicKey: publicKey, publicKeyInfo: leaf.RawSubjectPublicKeyInfo}, nil
}

func (p *PairingAnswerer) refuse(
	ctx context.Context,
	message *nats.Msg,
	pairing pairingoffers.OngoingPairing,
	refusal *pairingRefusal,
) {
	slog.WarnContext(ctx, "pairing request refused",
		slog.String("deviceName", string(pairing.DeviceName)),
		slog.String("reason", string(refusal.reason)),
		slog.Any("cause", refusal.cause),
	)
	p.offers.Refuse(ctx, companionprotocol.SecretIDFrom(pairing.Secret), refusal.reason)
	respond(ctx, message, companionprotocol.RefusedReply(refusal.reason))
}

func respond(ctx context.Context, message *nats.Msg, reply any) {
	data, err := json.Marshal(reply)
	if err == nil {
		err = message.Respond(data)
	}
	if err != nil {
		slog.WarnContext(ctx, "companion reply not sent",
			slog.String("subject", message.Subject), slog.Any("error", err),
		)
	}
}

func (p *PairingAnswerer) accept(
	ctx context.Context,
	message *nats.Msg,
	pairing pairingoffers.OngoingPairing,
	request provenRequest,
) {
	p.devices.Add(ctx, pairedDeviceFrom(pairing.DeviceName, request, p.clock.Now()))
	p.offers.Accept(ctx, companionprotocol.SecretIDFrom(pairing.Secret), request.deviceID)
	respond(ctx, message, companionprotocol.AcceptedReply(request.deviceID))
}

func pairedDeviceFrom(
	deviceName companionprotocol.DeviceName,
	request provenRequest,
	pairedAt time.Time,
) paireddevices.Device {
	return paireddevices.Device{
		ID:         request.deviceID,
		Name:       deviceName,
		PairedAt:   pairedAt,
		AppVersion: request.appVersion,
	}
}
