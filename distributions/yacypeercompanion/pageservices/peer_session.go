package pageservices

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
)

const peerAnswerWait = 10 * time.Second

var (
	errAppOutdated  = errors.New("the peer finds the app outdated")
	errPeerOutdated = errors.New("the peer is outdated for the app")
	errHelloRefused = errors.New("the peer refused the hello")
)

type sessionTerms struct {
	link         companionprotocol.PairLink
	brokerAccess jetstreamconnect.BrokerAccess
	deviceID     companionprotocol.DeviceID
	appVersion   string
}

type peerSession struct {
	connection *nats.Conn
	deviceID   companionprotocol.DeviceID
	settings   companionprotocol.DeviceSettings
}

func (t sessionTerms) openSession(onRefused func()) (*peerSession, error) {
	connection, err := brokeraccess.NewConnector(wallclock.Clock{}).
		ConnectWithReconnects(t.link, t.brokerAccess, onRefused)
	if err != nil {
		return nil, err
	}
	if err := t.sayHelloOver(connection); err != nil {
		connection.Close()
		return nil, err
	}
	settings, err := settingsOver(connection, t.deviceID)
	if err != nil {
		connection.Close()
		return nil, err
	}
	return &peerSession{connection: connection, deviceID: t.deviceID, settings: settings}, nil
}

func (t sessionTerms) sayHelloOver(connection *nats.Conn) error {
	hello, _ := json.Marshal(companionprotocol.Hello{ //nolint:errchkjson // a string and an int.
		AppVersion:      t.appVersion,
		ProtocolVersion: companionprotocol.ProtocolVersion,
	})
	reply, err := connection.Request(
		companionprotocol.HelloSubjectOf(t.deviceID), hello, peerAnswerWait,
	)
	if err != nil {
		return fmt.Errorf("say hello: %w", err)
	}
	var helloReply companionprotocol.HelloReply
	if err := json.Unmarshal(reply.Data, &helloReply); err != nil {
		return fmt.Errorf("read the hello reply: %w", err)
	}
	return refusalOf(helloReply)
}

func refusalOf(reply companionprotocol.HelloReply) error {
	switch {
	case reply.Outcome == companionprotocol.OutcomeAccepted:
		return nil
	case reply.Reason == companionprotocol.ReasonAppOutdated:
		return errAppOutdated
	case reply.Reason == companionprotocol.ReasonPeerOutdated:
		return errPeerOutdated
	}
	return fmt.Errorf("%w: %s", errHelloRefused, reply.Reason)
}

func settingsOver(
	connection *nats.Conn,
	deviceID companionprotocol.DeviceID,
) (companionprotocol.DeviceSettings, error) {
	reply, err := connection.Request(
		companionprotocol.SettingsSubjectOf(deviceID), nil, peerAnswerWait,
	)
	if err != nil {
		return companionprotocol.DeviceSettings{}, fmt.Errorf("ask for the settings: %w", err)
	}
	var settings companionprotocol.DeviceSettings
	if err := json.Unmarshal(reply.Data, &settings); err != nil {
		return companionprotocol.DeviceSettings{}, fmt.Errorf("read the settings: %w", err)
	}
	return settings, nil
}

func (t sessionTerms) probeRefusal() bool {
	connection, err := brokeraccess.NewConnector(wallclock.Clock{}).
		ConnectOnce(t.link, t.brokerAccess)
	if err != nil {
		return errors.Is(err, nats.ErrAuthorization)
	}
	connection.Close()
	return false
}

func (s *peerSession) brokerURL() string {
	return s.connection.ConnectedUrl()
}

func (s *peerSession) partitionExponent() uint {
	return s.settings.PartitionExponent
}

func (s *peerSession) close() {
	s.connection.Close()
}
