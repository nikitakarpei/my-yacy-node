package companionspage

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
)

const (
	fieldCompanionsFetch = "companionsFetch"
	fieldPeerFillsIn     = "peerFillsIn"
)

type settingsView struct {
	Frame            pageFrame
	Header           headerView
	DeviceStatesPath string
	CompanionsPath   string
	AssetsPath       string
	SettingsPath     string
	Choice           companionsettings.Choice
	FieldNames       settingsFieldNames
}

type settingsFieldNames struct {
	CompanionsFetch string
	PeerFillsIn     string
}

func (p *pages) settings(w http.ResponseWriter, r *http.Request) {
	rows := p.deviceRows()
	render(w, r, settingsPage, settingsView{
		Frame:            p.frame,
		Header:           p.headerFor(rows, settingsTab),
		DeviceStatesPath: pathDeviceStates,
		CompanionsPath:   Path,
		AssetsPath:       pathAssets,
		SettingsPath:     pathSettings,
		Choice:           p.companionSettings.Choice(),
		FieldNames: settingsFieldNames{
			CompanionsFetch: fieldCompanionsFetch,
			PeerFillsIn:     fieldPeerFillsIn,
		},
	})
}

func (p *pages) choose(w http.ResponseWriter, r *http.Request) {
	p.companionSettings.Choose(r.Context(), p.choiceFrom(r))
	http.Redirect(w, r, pathSettings, http.StatusSeeOther)
}

func (p *pages) choiceFrom(r *http.Request) companionsettings.Choice {
	choice := p.companionSettings.Choice()
	if choice.CompanionsFetch {
		choice.PeerFillsIn = r.PostFormValue(fieldPeerFillsIn) != ""
	}
	choice.CompanionsFetch = r.PostFormValue(fieldCompanionsFetch) != ""
	return choice
}
