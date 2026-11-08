package paireddevices_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
)

var pairedAt = time.Date(2026, 11, 7, 14, 2, 11, 0, time.UTC)

func openDevices(t *testing.T, directory string) *paireddevices.Devices {
	t.Helper()
	devices, err := paireddevices.Open(directory)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(devices.Close)
	return devices
}

func pixel() paireddevices.Device {
	return paireddevices.Device{
		ID: "device-a", Name: "Pixel 8", PairedAt: pairedAt, AppVersion: "1.2.0",
	}
}

func TestAPairedDeviceStaysPairedAcrossStarts(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "companions")
	openDevices(t, directory).Add(context.Background(), pixel())

	reopened := openDevices(t, directory)

	if !reopened.Holds("device-a") {
		t.Fatal("the device is not paired after a restart")
	}
	if got := reopened.List(); len(got) != 1 || got[0] != pixel() {
		t.Errorf("devices = %+v, want %+v", got, pixel())
	}
}

func TestAnUnpairedDeviceIsGoneAcrossStarts(t *testing.T) {
	directory := t.TempDir()
	devices := openDevices(t, directory)
	devices.Add(context.Background(), pixel())

	devices.Remove(context.Background(), "device-a")

	if devices.Holds("device-a") || openDevices(t, directory).Holds("device-a") {
		t.Error("the unpaired device is still paired")
	}
}

func TestPairingADeviceAgainReplacesIt(t *testing.T) {
	devices := openDevices(t, t.TempDir())
	devices.Add(context.Background(), pixel())
	renamed := pixel()
	renamed.Name = "Work phone"

	devices.Add(context.Background(), renamed)

	if got := devices.List(); len(got) != 1 || got[0].Name != "Work phone" {
		t.Errorf("devices = %+v, want one device named Work phone", got)
	}
}

func TestTheDeviceKeepsItsLastConnectionAndAppVersion(t *testing.T) {
	directory := t.TempDir()
	devices := openDevices(t, directory)
	devices.Add(context.Background(), pixel())
	connectedAt := pairedAt.Add(time.Hour)

	devices.RecordConnection(context.Background(), "device-a", connectedAt)
	devices.RecordAppVersion(context.Background(), "device-a", "1.3.0")
	devices.RecordAppVersion(context.Background(), "device-b", "9.9.9")

	got := openDevices(t, directory).List()
	if len(got) != 1 || !got[0].LastConnectedAt.Equal(connectedAt) || got[0].AppVersion != "1.3.0" {
		t.Errorf("devices = %+v, want the connection and the app version kept", got)
	}
}

func TestOnlyTheOwnerCanReadThePairedDevices(t *testing.T) {
	directory := t.TempDir()
	openDevices(t, directory).Add(context.Background(), pixel())

	info, err := os.Stat(filepath.Join(directory, "paired-devices.json"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestOpenRefusesAnUnreadableFile(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "paired-devices.json"), []byte("{"), 0o600,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := paireddevices.Open(directory); err == nil {
		t.Error("want an error for an unreadable file")
	}
}
