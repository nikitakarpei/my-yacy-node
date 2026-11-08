package companionspage_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

const eventWait = 5 * time.Second

var lastConnectedAt = time.Date(2026, 11, 5, 22, 14, 0, 0, time.Local)

type fixture struct {
	devices   []paireddevices.Device
	connected map[companionprotocol.DeviceID]bool
	unpaired  []companionprotocol.DeviceID
	offer     *fakeOffer
	mu        sync.Mutex
}

func (f *fixture) List() []paireddevices.Device { return f.devices }

func (f *fixture) ConnectedDevices() map[companionprotocol.DeviceID]bool { return f.connected }

func (f *fixture) Unpair(_ context.Context, deviceID companionprotocol.DeviceID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unpaired = append(f.unpaired, deviceID)
}

func (f *fixture) Open() companionspage.PairingOffer { return f.offer }

type fakeOffer struct {
	events chan pairingoffers.Event
	closed chan struct{}
}

func (o *fakeOffer) Events() <-chan pairingoffers.Event { return o.events }

func (o *fakeOffer) Close() { close(o.closed) }

func serve(t *testing.T, f *fixture) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	companionspage.Register(mux, companionspage.Collaborators{
		Devices: f, Unpairer: f, Offers: f, DeviceConnections: f,
	}, companionprotocol.PairLink{
		Addresses: []string{"192.168.1.20:4223"}, BrokerKeyHash: "hash",
	}, webuistyle.TabBar{})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func pageOf(t *testing.T, address string) string {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", address, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %d %v", address, response.StatusCode, err)
	}
	return string(body)
}

func pairedFixture() *fixture {
	return &fixture{
		devices: []paireddevices.Device{
			{ID: "device-b", Name: "Pixel 8", AppVersion: "1.2.0"},
			{
				ID: "device-a", Name: "Galaxy Tab S9", AppVersion: "1.1.0",
				LastConnectedAt: lastConnectedAt,
			},
		},
		connected: map[companionprotocol.DeviceID]bool{"device-b": true},
	}
}

func TestTheCompanionsPageListsEachPairedDevice(t *testing.T) {
	page := pageOf(t, serve(t, pairedFixture()).URL+companionspage.Path)

	for _, want := range []string{
		"Pixel 8", "1.2.0", "Connected", "Galaxy Tab S9", "1.1.0", "5 Nov 2026, 22:14",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Index(page, "Galaxy Tab S9") > strings.Index(page, "Pixel 8") {
		t.Error("the devices are not in the order of their names")
	}
	if strings.Contains(page, "No companions are paired.") {
		t.Error("the page says no companions are paired")
	}
}

func TestTheCompanionsPageSaysWhenNoDeviceIsPaired(t *testing.T) {
	page := pageOf(t, serve(t, &fixture{}).URL+companionspage.Path)

	if !strings.Contains(page, "No companions are paired.") {
		t.Error("the page does not say that no companions are paired")
	}
}

func TestTheCompanionsPageNamesTheDeviceThatJustPaired(t *testing.T) {
	page := pageOf(t, serve(t, pairedFixture()).URL+companionspage.Path+"?paired=device-b")

	if !strings.Contains(page, "Pixel 8 is paired.") {
		t.Error("the page does not name the device that just paired")
	}
}

func TestTheCompanionsPageAsksBeforeItUnpairsADevice(t *testing.T) {
	page := pageOf(t, serve(t, pairedFixture()).URL+companionspage.Path+"?unpair=device-a")

	if !strings.Contains(page, "Unpair Galaxy Tab S9?") {
		t.Error("the page does not ask to unpair the device")
	}
}

func TestUnpairingADeviceGoesBackToTheList(t *testing.T) {
	f := pairedFixture()
	server := serve(t, f)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		server.URL+companionspage.Path+"/unpair",
		strings.NewReader(url.Values{"device": {"device-a"}}.Encode()))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusSeeOther ||
		response.Header.Get("Location") != companionspage.Path {
		t.Errorf("response = %d to %q, want 303 to the list",
			response.StatusCode, response.Header.Get("Location"))
	}
	if len(f.unpaired) != 1 || f.unpaired[0] != "device-a" {
		t.Errorf("unpaired = %v, want device-a", f.unpaired)
	}
}

func TestThePairPageShowsTheSteps(t *testing.T) {
	page := pageOf(t, serve(t, &fixture{}).URL+companionspage.Path+"/pair")

	for _, want := range []string{
		"Connect the Android device to the same network as the peer.",
		"YaCy Companion app",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

type serverEvents struct {
	lines *bufio.Scanner
	body  io.Closer
}

func openEvents(t *testing.T, address string) serverEvents {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type = %q", response.Header.Get("Content-Type"))
	}
	return serverEvents{lines: bufio.NewScanner(response.Body), body: response.Body}
}

func (e serverEvents) next(t *testing.T) (string, string) {
	t.Helper()
	var name, data string
	for e.lines.Scan() {
		line := e.lines.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
		case line == "" && name != "":
			return name, data
		}
	}
	t.Fatalf("the event stream ended: %v", e.lines.Err())
	return "", ""
}

func TestThePairPageHearsEachStepOfAPairing(t *testing.T) {
	offer := &fakeOffer{events: make(chan pairingoffers.Event, 8), closed: make(chan struct{})}
	server := serve(t, &fixture{offer: offer})
	events := openEvents(t, server.URL+companionspage.Path+"/pair/events")

	offer.events <- pairingoffers.SecretRenewed{Secret: "secret"}
	if name, data := events.next(t); name != "code" || !strings.HasPrefix(data, "<svg") {
		t.Errorf("event = %s %q, want a code as SVG", name, data)
	}
	offer.events <- pairingoffers.PairingBegun{DeviceName: "Pixel 8"}
	if name, data := events.next(t); name != "begun" || !strings.Contains(data, "Pixel 8") {
		t.Errorf("event = %s %q, want started by Pixel 8", name, data)
	}
	offer.events <- pairingoffers.DeviceRefused{
		DeviceName: "Pixel 8", Reason: companionprotocol.ReasonAppOutdated,
	}
	if name, data := events.next(t); name != "refused" ||
		!strings.Contains(data, "Please update the app.") {
		t.Errorf("event = %s %q, want refused as outdated", name, data)
	}
	offer.events <- pairingoffers.DevicePaired{DeviceName: "Pixel 8", DeviceID: "device-b"}
	if name, data := events.next(t); name != "paired" ||
		!strings.Contains(data, companionspage.Path+"?paired=device-b") {
		t.Errorf("event = %s %q, want paired with the list address", name, data)
	}

	select {
	case <-offer.closed:
	case <-time.After(eventWait):
		t.Error("the offer stays open after the pairing")
	}
}

func TestEachRefusalHasAReason(t *testing.T) {
	for _, reason := range []companionprotocol.RefusalReason{
		companionprotocol.ReasonAppOutdated,
		companionprotocol.ReasonPeerOutdated,
		companionprotocol.ReasonAppNotOfficial,
		companionprotocol.ReasonDeviceNotTrusted,
		companionprotocol.ReasonRequestInvalid,
	} {
		offer := &fakeOffer{events: make(chan pairingoffers.Event, 1), closed: make(chan struct{})}
		events := openEvents(t, serve(t, &fixture{offer: offer}).URL+companionspage.Path+
			"/pair/events")

		offer.events <- pairingoffers.DeviceRefused{DeviceName: "Pixel 8", Reason: reason}

		_, data := events.next(t)
		var refusal struct {
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(data), &refusal); err != nil || refusal.Reason == "" {
			t.Errorf("refusal for %s = %q (%v), want a reason", reason, data, err)
		}
	}
}

func TestTheStatesOfThePairedDevicesReadAsJSON(t *testing.T) {
	body := pageOf(t, serve(t, pairedFixture()).URL+companionspage.Path+"/device-states")

	var devices []struct {
		ID            string `json:"id"`
		AppVersion    string `json:"appVersion"`
		LastConnected string `json:"lastConnected"`
		Connected     bool   `json:"connected"`
	}
	if err := json.Unmarshal([]byte(body), &devices); err != nil {
		t.Fatalf("Unmarshal %s: %v", body, err)
	}
	if len(devices) != 2 || devices[0].ID != "device-a" ||
		devices[0].LastConnected != "5 Nov 2026, 22:14" || devices[0].Connected ||
		devices[1].ID != "device-b" || !devices[1].Connected || devices[1].AppVersion != "1.2.0" {
		t.Errorf("devices = %+v, want both devices with their states", devices)
	}
}
