package companionspage_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
)

const pausedText = "Page fetching is paused: no companion is connected."

type redirect struct {
	status   int
	location string
}

func changeFetching(t *testing.T, server string, form string) redirect {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		server+companionspage.Path+"/settings", strings.NewReader(form))
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

func TestTheSettingsPageShowsTheFetching(t *testing.T) {
	f := pairedFixture()
	f.fetching = companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}

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
	f.fetching = companionsettings.Fetching{PeerFillsIn: true}

	page := pageOf(t, serve(t, f).URL+companionspage.Path+"/settings")

	if field := switchOf(page, "peerFillsIn"); !strings.Contains(field, "checked") ||
		!strings.Contains(field, "disabled") {
		t.Errorf("fill-in switch = %q, want on and disabled", field)
	}
}

func TestTurningCompanionsOnKeepsTheFetchingAndGoesBackToTheSettings(t *testing.T) {
	f := pairedFixture()

	response := changeFetching(t, serve(t, f).URL, url.Values{"companionsFetch": {"on"}}.Encode())

	if response.status != http.StatusSeeOther ||
		response.location != companionspage.Path+"/settings" {
		t.Errorf(
			"response = %d to %q, want 303 to the settings",
			response.status,
			response.location,
		)
	}
	if f.Fetching() != (companionsettings.Fetching{CompanionsFetch: true}) {
		t.Errorf("fetching = %+v, want companions fetching", f.Fetching())
	}
}

func TestTurningCompanionsOffKeepsTheFillIn(t *testing.T) {
	f := pairedFixture()
	f.fetching = companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}

	changeFetching(t, serve(t, f).URL, url.Values{"peerFillsIn": {"on"}}.Encode())

	if f.Fetching() != (companionsettings.Fetching{PeerFillsIn: true}) {
		t.Errorf("fetching = %+v, want only the fill-in kept", f.Fetching())
	}
}

func TestTheFillInStaysWhileCompanionsAreOff(t *testing.T) {
	f := pairedFixture()
	f.fetching = companionsettings.Fetching{PeerFillsIn: true}

	changeFetching(t, serve(t, f).URL, url.Values{"companionsFetch": {"on"}}.Encode())

	if f.Fetching() != (companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}) {
		t.Errorf("fetching = %+v, want companions fetching with the kept fill-in", f.Fetching())
	}
}

func TestTheFillInTurnsOnAndOffWhileCompanionsFetch(t *testing.T) {
	for _, change := range []struct {
		name   string
		stored companionsettings.Fetching
		form   url.Values
		want   companionsettings.Fetching
	}{
		{
			name:   "on",
			stored: companionsettings.Fetching{CompanionsFetch: true},
			form:   url.Values{"companionsFetch": {"on"}, "peerFillsIn": {"on"}},
			want:   companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true},
		},
		{
			name:   "off",
			stored: companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true},
			form:   url.Values{"companionsFetch": {"on"}},
			want:   companionsettings.Fetching{CompanionsFetch: true},
		},
	} {
		t.Run(change.name, func(t *testing.T) {
			f := pairedFixture()
			f.fetching = change.stored

			changeFetching(t, serve(t, f).URL, change.form.Encode())

			if f.Fetching() != change.want {
				t.Errorf("fetching = %+v, want %+v", f.Fetching(), change.want)
			}
		})
	}
}

func TestAnUnreadableSettingsFormChangesNothing(t *testing.T) {
	f := pairedFixture()
	f.fetching = companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}

	response := changeFetching(t, serve(t, f).URL, "companionsFetch=%zz")

	if response.status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", response.status)
	}
	if f.Fetching() != (companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}) {
		t.Errorf("fetching = %+v, want it unchanged", f.Fetching())
	}
}

func pausedBannerOf(page string) string {
	start := strings.Index(page, `id="paused-banner"`)
	if start < 0 {
		return ""
	}
	return page[start : start+strings.Index(page[start:], ">")]
}

func TestBothTabsSayWhenPageFetchingIsPaused(t *testing.T) {
	f := pairedFixture()
	f.fetchingPaused = true
	server := serve(t, f).URL

	for _, path := range []string{companionspage.Path, companionspage.Path + "/settings"} {
		page := pageOf(t, server+path)
		if banner := pausedBannerOf(page); banner == "" || strings.Contains(banner, "hidden") ||
			!strings.Contains(page, pausedText) {
			t.Errorf("%s does not say that page fetching is paused", path)
		}
	}
}

func TestNeitherTabSaysPausedWhilePagesAreFetched(t *testing.T) {
	server := serve(t, pairedFixture()).URL

	for _, path := range []string{companionspage.Path, companionspage.Path + "/settings"} {
		if banner := pausedBannerOf(pageOf(t, server+path)); !strings.Contains(banner, "hidden") {
			t.Errorf("%s says that page fetching is paused", path)
		}
	}
}
