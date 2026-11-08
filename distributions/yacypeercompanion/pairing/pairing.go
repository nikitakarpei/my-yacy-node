// Package pairing pairs the app with a peer from the text of a pairing code, and unpairs it. It
// tells which kind of link a text is, gives the challenge for the attestation of the device key,
// sends the pairing request with the secret of the link, and reads the outcome.
package pairing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
)

const (
	LinkPair       = "pair"
	LinkUnreadable = "unreadable"
	LinkForeign    = "foreign"

	ResultAccepted         = "accepted"
	ResultNoAnswer         = "no-answer"
	ResultSecretExpired    = "secret-expired"
	ResultAppOutdated      = "app-outdated"
	ResultPeerOutdated     = "peer-outdated"
	ResultAppNotOfficial   = "app-not-official"
	ResultDeviceNotTrusted = "device-not-trusted"
	ResultRefused          = "refused"

	replyWait = 15 * time.Second
)

type DeviceKey interface {
	PublicKeyInfo() []byte
	SignDigest(digest []byte) ([]byte, error)
}

type Outcome struct {
	Result string
	Device string
}

func LinkKindOf(text string) string {
	_, err := companionprotocol.PairLinkFrom(text)
	switch {
	case err == nil:
		return LinkPair
	case errors.Is(err, companionprotocol.ErrUnreadableLink):
		return LinkUnreadable
	default:
		return LinkForeign
	}
}

func ChallengeOf(linkText string) []byte {
	link, err := companionprotocol.PairLinkFrom(linkText)
	if err != nil {
		slog.WarnContext(context.Background(), "pairing challenge not made", slog.Any("error", err))
		return nil
	}
	return companionprotocol.ChallengeFor(link.Secret, link.BrokerKeyHash)
}

type Request struct {
	linkText    string
	deviceName  string
	appVersion  string
	attestation [][]byte
}

func NewRequest(linkText string, deviceName string, appVersion string) *Request {
	return &Request{linkText: linkText, deviceName: deviceName, appVersion: appVersion}
}

func (r *Request) AddCertificate(der []byte) {
	r.attestation = append(r.attestation, bytes.Clone(der))
}

func (r *Request) Send(key DeviceKey) *Outcome {
	link, err := companionprotocol.PairLinkFrom(r.linkText)
	if err != nil {
		slog.WarnContext(context.Background(), "pairing link unreadable", slog.Any("error", err))
		return &Outcome{Result: ResultRefused}
	}
	connection, err := brokeraccess.Connect(
		link,
		brokeraccess.PairingOptions(link, companionprotocol.DeviceName(r.deviceName)),
	)
	if errors.Is(err, nats.ErrAuthorization) {
		return &Outcome{Result: ResultSecretExpired}
	}
	if err != nil {
		slog.WarnContext(
			context.Background(),
			"peer unreachable for pairing",
			slog.Any("error", err),
		)
		return &Outcome{Result: ResultNoAnswer}
	}
	defer connection.Close()
	data, err := r.requestDataFor(link, key)
	if err != nil {
		slog.WarnContext(context.Background(), "pairing request not made", slog.Any("error", err))
		return &Outcome{Result: ResultRefused}
	}
	reply, err := connection.Request(companionprotocol.PairingRequestSubject, data, replyWait)
	if err != nil {
		slog.WarnContext(context.Background(), "no pairing reply", slog.Any("error", err))
		return &Outcome{Result: ResultNoAnswer}
	}
	return outcomeOf(reply.Data)
}

func (r *Request) requestDataFor(link companionprotocol.PairLink, key DeviceKey) ([]byte, error) {
	signer, err := brokeraccess.SignerOf(key)
	if err != nil {
		return nil, err
	}
	proof, err := companionprotocol.ProofOf(
		signer, companionprotocol.ChallengeFor(link.Secret, link.BrokerKeyHash),
	)
	if err != nil {
		return nil, err
	}
	return json.Marshal(companionprotocol.PairingRequest{ //nolint:wrapcheck // plain types.
		Protocol:    companionprotocol.ProtocolVersion,
		AppVersion:  r.appVersion,
		Attestation: r.attestation,
		Proof:       proof,
	})
}

func outcomeOf(data []byte) *Outcome {
	var reply companionprotocol.PairingReply
	if err := json.Unmarshal(data, &reply); err != nil {
		slog.WarnContext(context.Background(), "pairing reply unreadable", slog.Any("error", err))
		return &Outcome{Result: ResultRefused}
	}
	if reply.Outcome == companionprotocol.OutcomeAccepted {
		return &Outcome{Result: ResultAccepted, Device: string(reply.Device)}
	}
	return &Outcome{Result: resultOf(reply.Reason)}
}

func resultOf(reason companionprotocol.Reason) string {
	switch reason {
	case companionprotocol.ReasonAppOutdated:
		return ResultAppOutdated
	case companionprotocol.ReasonPeerOutdated:
		return ResultPeerOutdated
	case companionprotocol.ReasonAppNotOfficial:
		return ResultAppNotOfficial
	case companionprotocol.ReasonDeviceNotTrusted:
		return ResultDeviceNotTrusted
	case companionprotocol.ReasonRequestInvalid:
		return ResultRefused
	}
	return ResultRefused
}
