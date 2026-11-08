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
	return answerUntilDone(ctx, connection, map[string]nats.MsgHandler{
		companionprotocol.PairingRequestSubject: func(message *nats.Msg) {
			p.answer(ctx, message)
		},
	})
}

func (p *PairingRequests) answer(ctx context.Context, message *nats.Msg) {
	pairing, held := p.offers.HoldForVerdict(
		ctx,
		companionprotocol.SecretIDFromReply(message.Reply),
	)
	if !held {
		slog.WarnContext(ctx, "pairing request left unanswered",
			slog.String("replySubject", message.Reply),
		)
		return
	}
	request, refused := pairingRequestFrom(message.Data)
	if refused != nil {
		p.refuse(ctx, message, pairing, refused)
		return
	}
	key, refused := p.provenKeyOf(pairing, request)
	if refused != nil {
		p.refuse(ctx, message, pairing, refused)
		return
	}
	p.accept(ctx, message, pairing, pairedDeviceFrom(pairing, request, key, p.clock.Now()))
}

type refusal struct {
	reason companionprotocol.Reason
	cause  error
}

func pairingRequestFrom(data []byte) (companionprotocol.PairingRequest, *refusal) {
	var request companionprotocol.PairingRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return request, &refusal{
			reason: companionprotocol.ReasonRequestInvalid,
			cause:  fmt.Errorf("read the pairing request: %w", err),
		}
	}
	return request, nil
}

func (p *PairingRequests) refuse(
	ctx context.Context,
	message *nats.Msg,
	pairing pairingoffers.OngoingPairing,
	refused *refusal,
) {
	slog.WarnContext(ctx, "pairing request refused",
		slog.String("deviceName", string(pairing.DeviceName)),
		slog.String("reason", string(refused.reason)),
		slog.Any("cause", refused.cause),
	)
	p.offers.Refuse(ctx, companionprotocol.SecretIDOf(pairing.Secret), refused.reason)
	respond(ctx, message, companionprotocol.RefusedReply(refused.reason))
}

func (p *PairingRequests) provenKeyOf(
	pairing pairingoffers.OngoingPairing,
	request companionprotocol.PairingRequest,
) (attestedKey, *refusal) {
	switch {
	case request.Protocol < companionprotocol.ProtocolVersion:
		return attestedKey{}, &refusal{
			reason: companionprotocol.ReasonAppOutdated,
			cause:  errOlderProtocol,
		}
	case request.Protocol > companionprotocol.ProtocolVersion:
		return attestedKey{}, &refusal{
			reason: companionprotocol.ReasonPeerOutdated,
			cause:  errNewerProtocol,
		}
	}
	key, err := attestedKeyOf(request.Attestation)
	if err != nil {
		return attestedKey{}, &refusal{reason: companionprotocol.ReasonRequestInvalid, cause: err}
	}
	challenge := companionprotocol.ChallengeFor(pairing.Secret, p.brokerKeyHash)
	if !companionprotocol.IsProofOf(key.publicKey, challenge, request.Proof) {
		return attestedKey{}, &refusal{
			reason: companionprotocol.ReasonRequestInvalid,
			cause:  errProofInvalid,
		}
	}
	return key, nil
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

func pairedDeviceFrom(
	pairing pairingoffers.OngoingPairing,
	request companionprotocol.PairingRequest,
	key attestedKey,
	now time.Time,
) paireddevices.Device {
	return paireddevices.Device{
		ID:         companionprotocol.DeviceIDOf(key.keyInfo),
		Name:       pairing.DeviceName,
		PairedAt:   now,
		AppVersion: request.AppVersion,
	}
}

func (p *PairingRequests) accept(
	ctx context.Context,
	message *nats.Msg,
	pairing pairingoffers.OngoingPairing,
	device paireddevices.Device,
) {
	p.devices.Add(ctx, device)
	p.offers.Accept(ctx, companionprotocol.SecretIDOf(pairing.Secret), device.ID)
	respond(ctx, message, companionprotocol.AcceptedReply(device.ID))
}
