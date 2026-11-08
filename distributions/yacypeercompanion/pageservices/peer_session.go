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

const settingsWait = 10 * time.Second

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
	settings, err := settingsOver(connection, t.deviceID)
	if err != nil {
		connection.Close()
		return nil, err
	}
	return &peerSession{connection: connection, deviceID: t.deviceID, settings: settings}, nil
}

func settingsOver(
	connection *nats.Conn,
	deviceID companionprotocol.DeviceID,
) (companionprotocol.DeviceSettings, error) {
	reply, err := connection.Request(
		companionprotocol.SettingsSubjectOf(deviceID), nil, settingsWait,
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

func (s *peerSession) reportApp(appVersion string) error {
	report, _ := json.Marshal( //nolint:errchkjson // one string field.
		companionprotocol.AppReport{AppVersion: appVersion},
	)
	if err := s.connection.Publish(
		companionprotocol.AppReportSubjectOf(s.deviceID),
		report,
	); err != nil {
		return fmt.Errorf("report the app: %w", err)
	}
	return nil
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
