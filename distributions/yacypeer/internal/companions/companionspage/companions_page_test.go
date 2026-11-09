package companionspage_test

import (
	"bufio"
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

var lastConnectedAt = time.Date(2026, 11, 5, 22, 14, 0, 0, time.Local)

type fixture struct {
	devices        []paireddevices.Device
	connected      map[companionprotocol.DeviceID]bool
	unpaired       []companionprotocol.DeviceID
	offer          *fakeOffer
	fetching       companionsettings.Fetching
	fetchingPaused bool
	mu             sync.Mutex
}

func (f *fixture) List() []paireddevices.Device { return f.devices }

func (f *fixture) ConnectedDevices() map[companionprotocol.DeviceID]bool { return f.connected }

func (f *fixture) Unpair(_ context.Context, deviceID companionprotocol.DeviceID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unpaired = append(f.unpaired, deviceID)
}

func (f *fixture) Open() companionspage.PairingOffer { return f.offer }

func (f *fixture) Fetching() companionsettings.Fetching {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetching
}

func (f *fixture) ChangeFetching(_ context.Context, fetching companionsettings.Fetching) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fetching = fetching
}

func (f *fixture) FetchingPaused() bool { return f.fetchingPaused }

type fakeOffer struct {
	events chan pairingoffers.Event
	closed chan struct{}
}

func (o *fakeOffer) Events() <-chan pairingoffers.Event { return o.events }

func (o *fakeOffer) Close() { close(o.closed) }

func serve(t *testing.T, f *fixture) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	companionspage.Register(
		mux,
		companionspage.NewDevicesPage(f, f, f, f, webuistyle.TabBar{}),
		companionspage.NewSettingsPage(f, f, webuistyle.TabBar{}),
		companionspage.NewPairPage(f, companionprotocol.PairLink{
			Addresses: []string{"192.168.1.20:4223"}, BrokerKeyHash: "hash",
		}, webuistyle.TabBar{}),
	)
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
	if strings.Contains(page, "No devices are paired") {
		t.Error("the page says no devices are paired")
	}
}

func TestTheCompanionsPageSaysWhenNoDeviceIsPaired(t *testing.T) {
	page := pageOf(t, serve(t, &fixture{}).URL+companionspage.Path)

	if !strings.Contains(page, "No devices are paired") {
		t.Error("the page does not say that no devices are paired")
	}
}

func TestEachLinkToTheAppGuideOpensInANewTab(t *testing.T) {
	server := serve(t, &fixture{})
	for page, linkText := range map[string]string{
		companionspage.Path:           "Learn more",
		companionspage.Path + "/pair": "YaCy Companion app",
	} {
		attributes := regexp.MustCompile(`<a ([^>]*)>` + linkText).
			FindStringSubmatch(pageOf(t, server.URL+page))
		if attributes == nil {
			t.Errorf("%s has no %s link", page, linkText)
			continue
		}
		for _, want := range []string{`target="_blank"`, `rel="noopener"`} {
			if !strings.Contains(attributes[1], want) {
				t.Errorf("the %s link on %s lacks %s", linkText, page, want)
			}
		}
	}
}

func TestTheCompanionsPageOffersToPairADeviceOnce(t *testing.T) {
	for name, devices := range map[string]*fixture{"no device": {}, "a paired device": pairedFixture()} {
		page := pageOf(t, serve(t, devices).URL+companionspage.Path)

		if count := strings.Count(page, ">Pair new device</a>"); count != 1 {
			t.Errorf("with %s the page offers to pair a device %d times", name, count)
		}
	}
}

func TestThePairButtonFollowsTheDeviceList(t *testing.T) {
	page := pageOf(t, serve(t, pairedFixture()).URL+companionspage.Path)

	if strings.Index(page, ">Pair new device</a>") < strings.Index(page, "</table>") {
		t.Error("the pair button is not below the device list")
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

func newFakeOffer() *fakeOffer {
	return &fakeOffer{events: make(chan pairingoffers.Event, 8), closed: make(chan struct{})}
}

func pairingEventsOf(t *testing.T, server *httptest.Server) serverEvents {
	t.Helper()
	page := pageOf(t, server.URL+companionspage.Path+"/pair")
	return openEvents(t, server.URL+dataAttributeOf(t, page, "data-pairing-events"))
}

func dataAttributeOf(t *testing.T, page string, attribute string) string {
	t.Helper()
	match := regexp.MustCompile(attribute + `="([^"]*)"`).FindStringSubmatch(page)
	if match == nil {
		t.Fatalf("the page has no %s", attribute)
	}
	return html.UnescapeString(match[1])
}

type pairingEventData struct {
	DeviceName    string `json:"deviceName"`
	Reason        string `json:"reason"`
	CompanionsURL string `json:"companionsURL"`
}

func pairingEventDataOf(t *testing.T, data string) pairingEventData {
	t.Helper()
	var eventData pairingEventData
	if err := json.Unmarshal([]byte(data), &eventData); err != nil {
		t.Fatalf("Unmarshal %s: %v", data, err)
	}
	return eventData
}

func TestThePairPageHearsEachStepOfAPairing(t *testing.T) {
	f := pairedFixture()
	f.offer = newFakeOffer()
	server := serve(t, f)
	events := pairingEventsOf(t, server)

	f.offer.events <- pairingoffers.SecretRenewed{Secret: "secret"}
	if name, data := events.next(t); name != "code" || !strings.HasPrefix(data, "<svg") {
		t.Errorf("event = %s %q, want a code as SVG", name, data)
	}
	f.offer.events <- pairingoffers.PairingBegun{DeviceName: "Pixel 8"}
	if name, data := events.next(t); name != "begun" ||
		pairingEventDataOf(t, data).DeviceName != "Pixel 8" {
		t.Errorf("event = %s %q, want begun by Pixel 8", name, data)
	}
	f.offer.events <- pairingoffers.DeviceRefused{
		DeviceName: "Pixel 8", Reason: companionprotocol.ReasonAppOutdated,
	}
	if name, data := events.next(t); name != "refused" ||
		pairingEventDataOf(t, data).DeviceName != "Pixel 8" ||
		!strings.Contains(pairingEventDataOf(t, data).Reason, "Please update the app.") {
		t.Errorf("event = %s %q, want Pixel 8 refused as outdated", name, data)
	}
	f.offer.events <- pairingoffers.DevicePaired{DeviceName: "Pixel 8", DeviceID: "device-b"}
	name, data := events.next(t)
	if name != "paired" || pairingEventDataOf(t, data).DeviceName != "Pixel 8" {
		t.Errorf("event = %s %q, want Pixel 8 paired", name, data)
	}

	page := pageOf(t, server.URL+pairingEventDataOf(t, data).CompanionsURL)
	banner := strings.Index(page, "Pixel 8 is paired.")
	if banner < 0 {
		t.Error("the list that the paired event names does not name the device that just paired")
	}
	if tabs := strings.Index(page, `aria-label="Companions"`); banner > tabs {
		t.Error("the list names the device that just paired below the tabs")
	}
	<-f.offer.closed
}

func TestLeavingThePairPageEndsItsOffer(t *testing.T) {
	offer := newFakeOffer()
	events := pairingEventsOf(t, serve(t, &fixture{offer: offer}))

	if err := events.body.Close(); err != nil {
		t.Fatalf("leave the page: %v", err)
	}

	<-offer.closed
}

func TestTheStatesOfThePageReadAsJSON(t *testing.T) {
	f := pairedFixture()
	f.fetchingPaused = true
	server := serve(t, f)
	page := pageOf(t, server.URL+companionspage.Path)
	body := pageOf(t, server.URL+dataAttributeOf(t, page, "data-states"))

	var states struct {
		Devices []struct {
			ID            string `json:"id"`
			AppVersion    string `json:"appVersion"`
			LastConnected string `json:"lastConnected"`
			Connected     bool   `json:"connected"`
		} `json:"devices"`
		FetchingPaused bool `json:"fetchingPaused"`
	}
	if err := json.Unmarshal([]byte(body), &states); err != nil {
		t.Fatalf("Unmarshal %s: %v", body, err)
	}
	devices := states.Devices
	if len(devices) != 2 || devices[0].ID != "device-a" ||
		devices[0].LastConnected != "5 Nov 2026, 22:14" || devices[0].Connected ||
		devices[1].ID != "device-b" || !devices[1].Connected || devices[1].AppVersion != "1.2.0" {
		t.Errorf("devices = %+v, want both devices with their states", devices)
	}
	if !states.FetchingPaused {
		t.Error("the states do not say that page fetching is paused")
	}
}
