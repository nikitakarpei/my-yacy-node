package companionrequests_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
)

type deviceRecords struct {
	appVersions map[companionprotocol.DeviceID]string
	recorded    chan companionprotocol.DeviceID
	unpaired    chan companionprotocol.DeviceID
	mu          sync.Mutex
}

func (d *deviceRecords) RecordAppVersion(
	_ context.Context,
	deviceID companionprotocol.DeviceID,
	appVersion string,
) {
	d.mu.Lock()
	d.appVersions[deviceID] = appVersion
	d.mu.Unlock()
	d.recorded <- deviceID
}

func (d *deviceRecords) Unpair(_ context.Context, deviceID companionprotocol.DeviceID) {
	d.unpaired <- deviceID
}

func (d *deviceRecords) appVersionOf(deviceID companionprotocol.DeviceID) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.appVersions[deviceID]
}

func startDeviceAnswerer(t *testing.T) (*deviceRecords, string) {
	t.Helper()
	url := natstestserver.Start(t)
	records := &deviceRecords{
		appVersions: map[companionprotocol.DeviceID]string{},
		recorded:    make(chan companionprotocol.DeviceID, 4),
		unpaired:    make(chan companionprotocol.DeviceID, 4),
	}
	answerer := companionrequests.NewDeviceAnswerer(
		records, records, companionprotocol.DeviceSettings{PartitionExponent: 4},
	)
	subscribe(t, answerer.Subscribe, url)
	return records, url
}

func awaitDevice(
	t *testing.T,
	deviceIDs <-chan companionprotocol.DeviceID,
) companionprotocol.DeviceID {
	t.Helper()
	select {
	case deviceID := <-deviceIDs:
		return deviceID
	case <-time.After(replyWait):
		t.Fatal("no device was reached")
		return ""
	}
}

func TestADeviceGetsTheSettingsOfThePeer(t *testing.T) {
	_, url := startDeviceAnswerer(t)

	reply, err := connectTo(t, url).Request(
		companionprotocol.SettingsSubjectOf("device-a"), nil, replyWait,
	)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	var settings companionprotocol.DeviceSettings
	if err := json.Unmarshal(reply.Data, &settings); err != nil ||
		settings.PartitionExponent != 4 {
		t.Errorf("settings = %s, want partition exponent 4", reply.Data)
	}
}

func helloReplyTo(
	t *testing.T,
	url string,
	deviceID companionprotocol.DeviceID,
	hello []byte,
) companionprotocol.HelloReply {
	t.Helper()
	reply, err := connectTo(t, url).Request(
		companionprotocol.HelloSubjectOf(deviceID), hello, replyWait,
	)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	var helloReply companionprotocol.HelloReply
	if err := json.Unmarshal(reply.Data, &helloReply); err != nil {
		t.Fatalf("hello reply %s: %v", reply.Data, err)
	}
	return helloReply
}

func helloIn(t *testing.T, protocolVersion int) []byte {
	t.Helper()
	hello, err := json.Marshal(companionprotocol.Hello{
		AppVersion:      "1.3.0",
		ProtocolVersion: protocolVersion,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return hello
}

func TestTheHelloOfADeviceInTheProtocolOfThePeerIsAcceptedWithItsAppVersion(t *testing.T) {
	records, url := startDeviceAnswerer(t)

	reply := helloReplyTo(t, url, "device-a", helloIn(t, companionprotocol.ProtocolVersion))

	if reply != companionprotocol.AcceptedHelloReply() {
		t.Errorf("reply = %+v, want accepted", reply)
	}
	if recorded := awaitDevice(t, records.recorded); recorded != "device-a" ||
		records.appVersionOf("device-a") != "1.3.0" {
		t.Errorf(
			"recorded %s at %q, want device-a at 1.3.0",
			recorded,
			records.appVersionOf(recorded),
		)
	}
}

func TestAHelloThatThePeerCannotServeIsRefusedWithItsReason(t *testing.T) {
	for name, flaw := range map[string]struct {
		hello []byte
		want  companionprotocol.RefusalReason
	}{
		"older protocol": {
			hello: helloIn(t, companionprotocol.ProtocolVersion-1),
			want:  companionprotocol.ReasonAppOutdated,
		},
		"newer protocol": {
			hello: helloIn(t, companionprotocol.ProtocolVersion+1),
			want:  companionprotocol.ReasonPeerOutdated,
		},
		"unreadable": {
			hello: []byte("{"),
			want:  companionprotocol.ReasonRequestInvalid,
		},
	} {
		t.Run(name, func(t *testing.T) {
			records, url := startDeviceAnswerer(t)

			reply := helloReplyTo(t, url, "device-a", flaw.hello)

			if reply != companionprotocol.RefusedHelloReply(flaw.want) {
				t.Errorf("reply = %+v, want refused as %s", reply, flaw.want)
			}
			if records.appVersionOf("device-a") != "" {
				t.Error("the app version of a refused hello is recorded")
			}
		})
	}
}

func TestADeviceThatAsksToUnpairIsUnpaired(t *testing.T) {
	records, url := startDeviceAnswerer(t)

	if _, err := connectTo(t, url).Request(
		companionprotocol.UnpairingRequestSubjectOf("device-a"), nil, replyWait,
	); err != nil {
		t.Fatalf("Request: %v", err)
	}

	if unpaired := awaitDevice(t, records.unpaired); unpaired != "device-a" {
		t.Errorf("unpaired = %s, want device-a", unpaired)
	}
}
