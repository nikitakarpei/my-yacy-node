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
	unpaired    []companionprotocol.DeviceID
	mu          sync.Mutex
}

func (d *deviceRecords) RecordAppVersion(
	_ context.Context,
	id companionprotocol.DeviceID,
	appVersion string,
) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.appVersions[id] = appVersion
}

func (d *deviceRecords) Unpair(_ context.Context, device companionprotocol.DeviceID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.unpaired = append(d.unpaired, device)
}

func (d *deviceRecords) appVersionOf(id companionprotocol.DeviceID) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.appVersions[id]
}

func (d *deviceRecords) unpairedDevices() []companionprotocol.DeviceID {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]companionprotocol.DeviceID{}, d.unpaired...)
}

func startDeviceRequests(t *testing.T) (*deviceRecords, string) {
	t.Helper()
	url := natstestserver.Start(t)
	records := &deviceRecords{appVersions: map[companionprotocol.DeviceID]string{}}
	requests := companionrequests.NewDeviceAnswerer(
		records, records, companionprotocol.DeviceSettings{PartitionExponent: 4},
	)
	serve(t, requests.Serve, url, 3)
	return records, url
}

func TestADeviceGetsTheSettingsOfThePeer(t *testing.T) {
	_, url := startDeviceRequests(t)

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

func TestTheAppVersionThatADeviceReportsIsRecorded(t *testing.T) {
	records, url := startDeviceRequests(t)
	connection := connectTo(t, url)
	data, _ := json.Marshal(companionprotocol.AppReport{AppVersion: "1.3.0"})

	if err := connection.Publish(
		companionprotocol.AppSubjectOf("device-b"),
		[]byte("{"),
	); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := connection.Publish(companionprotocol.AppSubjectOf("device-a"), data); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	deadline := time.Now().Add(replyWait)
	for records.appVersionOf("device-a") != "1.3.0" {
		if time.Now().After(deadline) {
			t.Fatal("the app version is not recorded")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if records.appVersionOf("device-b") != "" {
		t.Error("an unreadable report is recorded")
	}
}

func TestADeviceThatAsksToUnpairIsUnpaired(t *testing.T) {
	records, url := startDeviceRequests(t)

	if _, err := connectTo(t, url).Request(
		companionprotocol.UnpairSubjectOf("device-a"), nil, replyWait,
	); err != nil {
		t.Fatalf("Request: %v", err)
	}

	deadline := time.Now().Add(replyWait)
	for len(records.unpairedDevices()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if unpaired := records.unpairedDevices(); len(unpaired) != 1 || unpaired[0] != "device-a" {
		t.Errorf("unpaired = %v, want device-a", unpaired)
	}
}
