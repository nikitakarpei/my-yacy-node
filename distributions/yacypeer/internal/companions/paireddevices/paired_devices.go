// Package paireddevices keeps the devices that are paired with the peer in one file of a
// directory: the name of each device, when it paired, the app version it last reported, and when
// it last connected. It reads the file once at start and writes it again on each change, and
// tells its observers when the first device pairs and when the last one is unpaired.
package paireddevices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

const (
	fileName          = "paired-devices.json"
	draftFileName     = fileName + ".draft"
	directoryFileMode = 0o700
	fileMode          = 0o600
)

type Device struct {
	PairedAt        time.Time                    `json:"pairedAt"`
	LastConnectedAt time.Time                    `json:"lastConnectedAt"`
	ID              companionprotocol.DeviceID   `json:"id"`
	Name            companionprotocol.DeviceName `json:"name"`
	AppVersion      string                       `json:"appVersion"`
}

type keptFile struct {
	Devices []Device `json:"devices"`
}

type Observer interface {
	FirstDevicePaired(ctx context.Context)
	LastDeviceUnpaired(ctx context.Context)
}

type Devices struct {
	root      *os.Root
	observers []Observer
	devices   []Device
	mu        sync.Mutex
}

func Open(ctx context.Context, directory string, observers ...Observer) (*Devices, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the paired devices directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the paired devices directory: %w", err)
	}
	devices, err := keptDevicesIn(root)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	keptDevices := &Devices{root: root, observers: observers, devices: devices}
	if len(devices) > 0 {
		keptDevices.tellFirstDevicePaired(ctx)
	}
	return keptDevices, nil
}

func keptDevicesIn(root *os.Root) ([]Device, error) {
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the paired devices: %w", err)
	}
	var file keptFile
	if err := json.Unmarshal(kept, &file); err != nil {
		return nil, fmt.Errorf("read the paired devices: %w", err)
	}
	return file.Devices, nil
}

func (d *Devices) List() []Device {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.devices)
}

func (d *Devices) Holds(id companionprotocol.DeviceID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.positionOf(id) >= 0
}

func (d *Devices) Add(ctx context.Context, device Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if position := d.positionOf(device.ID); position >= 0 {
		d.devices[position] = device
		d.keep(ctx)
		return
	}
	d.devices = append(d.devices, device)
	d.keep(ctx)
	if len(d.devices) == 1 {
		d.tellFirstDevicePaired(ctx)
	}
}

func (d *Devices) Remove(ctx context.Context, id companionprotocol.DeviceID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	position := d.positionOf(id)
	if position < 0 {
		slog.DebugContext(ctx, "device to remove is not paired", slog.String("device", string(id)))
		return
	}
	d.devices = slices.Delete(d.devices, position, position+1)
	d.keep(ctx)
	if len(d.devices) == 0 {
		d.tellLastDeviceUnpaired(ctx)
	}
}

func (d *Devices) tellFirstDevicePaired(ctx context.Context) {
	for _, observer := range d.observers {
		observer.FirstDevicePaired(ctx)
	}
}

func (d *Devices) tellLastDeviceUnpaired(ctx context.Context) {
	for _, observer := range d.observers {
		observer.LastDeviceUnpaired(ctx)
	}
}

func (d *Devices) RecordConnection(
	ctx context.Context,
	id companionprotocol.DeviceID,
	at time.Time,
) {
	d.change(ctx, id, func(device *Device) { device.LastConnectedAt = at })
}

func (d *Devices) RecordAppVersion(
	ctx context.Context,
	id companionprotocol.DeviceID,
	appVersion string,
) {
	d.change(ctx, id, func(device *Device) { device.AppVersion = appVersion })
}

func (d *Devices) change(ctx context.Context, id companionprotocol.DeviceID, apply func(*Device)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	position := d.positionOf(id)
	if position < 0 {
		slog.DebugContext(ctx, "device to change is not paired", slog.String("device", string(id)))
		return
	}
	apply(&d.devices[position])
	d.keep(ctx)
}

func (d *Devices) positionOf(id companionprotocol.DeviceID) int {
	return slices.IndexFunc(d.devices, func(device Device) bool { return device.ID == id })
}

func (d *Devices) keep(ctx context.Context) {
	encoded, err := json.MarshalIndent(keptFile{Devices: d.devices}, "", "  ")
	if err == nil {
		err = d.root.WriteFile(draftFileName, append(encoded, '\n'), fileMode)
	}
	if err == nil {
		err = d.root.Rename(draftFileName, fileName)
	}
	if err != nil {
		slog.ErrorContext(ctx, "paired devices not kept", slog.Any("error", err))
	}
}

func (d *Devices) Close() {
	if err := d.root.Close(); err != nil {
		slog.WarnContext(context.Background(), "paired devices directory not closed",
			slog.Any("error", err),
		)
	}
}
