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

type membershipTerms struct {
	link         companionprotocol.PairLink
	brokerAccess jetstreamconnect.BrokerAccess
	device       companionprotocol.DeviceID
	appVersion   string
}

type membership struct {
	connection *nats.Conn
	settings   companionprotocol.Settings
}

func (t membershipTerms) joined(unpaired func()) (*membership, error) {
	connection, err := brokeraccess.NewConnector(wallclock.Clock{}).
		Connect(t.link, append(t.brokerAccess.Options(),
			nats.ClosedHandler(func(closed *nats.Conn) {
				if errors.Is(closed.LastError(), nats.ErrAuthorization) {
					unpaired()
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
	return &membership{connection: connection, settings: settings}, nil
}

func (t membershipTerms) refused() bool {
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

func (t membershipTerms) settingsOver(connection *nats.Conn) (companionprotocol.Settings, error) {
	reply, err := connection.Request(
		companionprotocol.SettingsSubjectOf(t.device), nil, settingsWait,
	)
	if err != nil {
		return companionprotocol.Settings{}, fmt.Errorf("ask for the settings: %w", err)
	}
	var settings companionprotocol.Settings
	if err := json.Unmarshal(reply.Data, &settings); err != nil {
		return companionprotocol.Settings{}, fmt.Errorf("read the settings: %w", err)
	}
	return settings, nil
}

func (t membershipTerms) reportAppOver(connection *nats.Conn) error {
	report, _ := json.Marshal( //nolint:errchkjson // one string field.
		companionprotocol.AppReport{AppVersion: t.appVersion},
	)
	if err := connection.Publish(companionprotocol.AppSubjectOf(t.device), report); err != nil {
		return fmt.Errorf("report the app: %w", err)
	}
	return nil
}

func (m *membership) brokerURL() string {
	return m.connection.ConnectedUrl()
}

func (m *membership) leave() {
	m.connection.Close()
}
