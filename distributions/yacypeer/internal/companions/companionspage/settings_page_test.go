package companionspage_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/fetchsettings"
)

const pausedText = "Page fetching is paused: no companion is connected."

type redirect struct {
	status   int
	location string
}

func choose(t *testing.T, server string, form url.Values) redirect {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		server+companionspage.Path+"/settings", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	return redirect{status: response.StatusCode, location: response.Header.Get("Location")}
}

func switchOf(page, name string) string {
	start := strings.Index(page, `name="`+name+`"`)
	if start < 0 {
		return ""
	}
	return page[start : start+strings.Index(page[start:], ">")]
}

func TestTheSettingsPageShowsTheChoice(t *testing.T) {
	f := pairedFixture()
	f.choice = fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true}

	page := pageOf(t, serve(t, f).URL+companionspage.Path+"/settings")

	for _, name := range []string{"companionsFetch", "peerFillsIn"} {
		if field := switchOf(page, name); !strings.Contains(field, "checked") ||
			strings.Contains(field, "disabled") {
			t.Errorf("switch %s = %q, want on and enabled", name, field)
		}
	}
}

func TestTheFillInWaitsForCompanionsAndKeepsItsValue(t *testing.T) {
	f := pairedFixture()
	f.choice = fetchsettings.Choice{PeerFillsIn: true}

	page := pageOf(t, serve(t, f).URL+companionspage.Path+"/settings")

	if field := switchOf(page, "peerFillsIn"); !strings.Contains(field, "checked") ||
		!strings.Contains(field, "disabled") {
		t.Errorf("fill-in switch = %q, want on and disabled", field)
	}
}

func TestTheSettingsCanChangeBeforeACompanionIsPaired(t *testing.T) {
	f := &fixture{choice: fetchsettings.Choice{CompanionsFetch: true}}

	page := pageOf(t, serve(t, f).URL+companionspage.Path+"/settings")

	for _, name := range []string{"companionsFetch", "peerFillsIn"} {
		if strings.Contains(switchOf(page, name), "disabled") {
			t.Errorf("switch %s cannot change while no companion is paired", name)
		}
	}
}

func TestChoosingCompanionsKeepsTheChoiceAndGoesBackToTheSettings(t *testing.T) {
	f := pairedFixture()

	response := choose(t, serve(t, f).URL, url.Values{"companionsFetch": {"on"}})

	if response.status != http.StatusSeeOther ||
		response.location != companionspage.Path+"/settings" {
		t.Errorf(
			"response = %d to %q, want 303 to the settings",
			response.status,
			response.location,
		)
	}
	if f.Choice() != (fetchsettings.Choice{CompanionsFetch: true}) {
		t.Errorf("choice = %+v, want companions fetching", f.Choice())
	}
}

func TestTurningCompanionsOffKeepsTheFillIn(t *testing.T) {
	f := pairedFixture()
	f.choice = fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true}

	choose(t, serve(t, f).URL, url.Values{"peerFillsIn": {"on"}})

	if f.Choice() != (fetchsettings.Choice{PeerFillsIn: true}) {
		t.Errorf("choice = %+v, want only the fill-in kept", f.Choice())
	}
}

func TestTheFillInStaysWhileCompanionsAreOff(t *testing.T) {
	f := pairedFixture()
	f.choice = fetchsettings.Choice{PeerFillsIn: true}

	choose(t, serve(t, f).URL, url.Values{"companionsFetch": {"on"}})

	if f.Choice() != (fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true}) {
		t.Errorf("choice = %+v, want companions fetching with the kept fill-in", f.Choice())
	}
}

func TestTheSettingsChangeWhileNoDeviceIsPaired(t *testing.T) {
	f := &fixture{choice: fetchsettings.Choice{CompanionsFetch: true}}

	choose(t, serve(t, f).URL, url.Values{})

	if f.Choice() != (fetchsettings.Choice{}) {
		t.Errorf("choice = %+v, want companions off", f.Choice())
	}
}

func TestPageFetchingIsNotPausedWhileNoDeviceIsPaired(t *testing.T) {
	f := &fixture{choice: fetchsettings.Choice{CompanionsFetch: true}}

	page := pageOf(t, serve(t, f).URL+companionspage.Path)

	start := strings.Index(page, `id="paused"`)
	if !strings.Contains(page[start:start+strings.Index(page[start:], ">")], "hidden") {
		t.Error("the page says that page fetching is paused although no device is paired")
	}
}

func TestBothTabsSayWhenPageFetchingIsPaused(t *testing.T) {
	f := pairedFixture()
	f.connected = nil
	f.choice = fetchsettings.Choice{CompanionsFetch: true}
	server := serve(t, f).URL

	for _, path := range []string{companionspage.Path, companionspage.Path + "/settings"} {
		page := pageOf(t, server+path)
		start := strings.Index(page, `id="paused"`)
		if start < 0 || !strings.Contains(page, pausedText) ||
			strings.Contains(page[start:start+strings.Index(page[start:], ">")], "hidden") {
			t.Errorf("%s does not say that page fetching is paused", path)
		}
	}
}

func TestPageFetchingIsNotPausedWhileThePeerFillsIn(t *testing.T) {
	for name, f := range map[string]*fixture{
		"companion connected": func() *fixture {
			f := pairedFixture()
			f.choice = fetchsettings.Choice{CompanionsFetch: true}
			return f
		}(),
		"peer fills in": func() *fixture {
			f := pairedFixture()
			f.connected = nil
			f.choice = fetchsettings.Choice{CompanionsFetch: true, PeerFillsIn: true}
			return f
		}(),
	} {
		page := pageOf(t, serve(t, f).URL+companionspage.Path)
		start := strings.Index(page, `id="paused"`)
		if !strings.Contains(page[start:start+strings.Index(page[start:], ">")], "hidden") {
			t.Errorf("%s: the page says that page fetching is paused", name)
		}
	}
}
