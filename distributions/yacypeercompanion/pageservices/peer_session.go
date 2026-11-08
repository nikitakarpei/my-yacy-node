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
	device       companionprotocol.DeviceID
	appVersion   string
}

type deviceSession struct {
	connection *nats.Conn
	settings   companionprotocol.DeviceSettings
}

func (t sessionTerms) opened(refused func()) (*deviceSession, error) {
	connection, err := brokeraccess.NewConnector(wallclock.Clock{}).
		Connect(t.link, append(t.brokerAccess.Options(),
			nats.ClosedHandler(func(closed *nats.Conn) {
				if errors.Is(closed.LastError(), nats.ErrAuthorization) {
					refused()
				}
			}),
		))
	if err != nil {
		return nil, err
	}
	settings, err := t.settingsOver(connection)
	if err == nil {
		err = t.reportAppOver(connection)
	}
	if err != nil {
		connection.Close()
		return nil, err
	}
	return &deviceSession{connection: connection, settings: settings}, nil
}

func (t sessionTerms) refused() bool {
	connection, err := brokeraccess.NewConnector(wallclock.Clock{}).Connect(
		t.link,
		append(t.brokerAccess.Options(), nats.NoReconnect()),
	)
	if err != nil {
		return errors.Is(err, nats.ErrAuthorization)
	}
	connection.Close()
	return false
}

func (t sessionTerms) settingsOver(
	connection *nats.Conn,
) (companionprotocol.DeviceSettings, error) {
	reply, err := connection.Request(
		companionprotocol.SettingsSubjectOf(t.device), nil, settingsWait,
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

func (t sessionTerms) reportAppOver(connection *nats.Conn) error {
	report, _ := json.Marshal( //nolint:errchkjson // one string field.
		companionprotocol.AppReport{AppVersion: t.appVersion},
	)
	if err := connection.Publish(companionprotocol.AppSubjectOf(t.device), report); err != nil {
		return fmt.Errorf("report the app: %w", err)
	}
	return nil
}

func (s *deviceSession) brokerURL() string {
	return s.connection.ConnectedUrl()
}

func (s *deviceSession) close() {
	s.connection.Close()
}
