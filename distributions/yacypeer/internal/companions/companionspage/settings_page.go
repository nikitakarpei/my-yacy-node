package companionspage

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
)

const (
	fieldCompanionsFetch = "companionsFetch"
	fieldPeerFillsIn     = "peerFillsIn"
)

type settingsPage struct {
	settings      CompanionSettings
	fetchingPause FetchingPause
	frame         pageFrame
}

type settingsView struct {
	Frame        pageFrame
	Header       headerView
	StatesPath   string
	AssetsPath   string
	SettingsPath string
	Fetching     companionsettings.Fetching
	FieldNames   settingsFieldNames
}

type settingsFieldNames struct {
	CompanionsFetch string
	PeerFillsIn     string
}

func (s *settingsPage) show(w http.ResponseWriter, r *http.Request) {
	render(w, r, settingsTemplate, settingsView{
		Frame:        s.frame,
		Header:       headerFor(s.fetchingPause.FetchingPaused(), settingsTab),
		StatesPath:   pathStates,
		AssetsPath:   pathAssets,
		SettingsPath: pathSettings,
		Fetching:     s.settings.Fetching(),
		FieldNames: settingsFieldNames{
			CompanionsFetch: fieldCompanionsFetch,
			PeerFillsIn:     fieldPeerFillsIn,
		},
	})
}

func (s *settingsPage) change(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.WarnContext(r.Context(), "companion settings form unreadable", slog.Any("error", err))
		http.Error(w, "companion settings form unreadable", http.StatusBadRequest)
		return
	}
	s.settings.ChangeFetching(r.Context(), s.fetchingFrom(r.PostForm))
	http.Redirect(w, r, pathSettings, http.StatusSeeOther)
}

func (s *settingsPage) fetchingFrom(form url.Values) companionsettings.Fetching {
	fetching := s.settings.Fetching()
	if fetching.CompanionsFetch {
		fetching.PeerFillsIn = form.Get(fieldPeerFillsIn) != ""
	}
	fetching.CompanionsFetch = form.Get(fieldCompanionsFetch) != ""
	return fetching
}
