package companionspage

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/fetchsettings"
)

const (
	fieldCompanionsFetch = "companionsFetch"
	fieldPeerFillsIn     = "peerFillsIn"
)

type settingsView struct {
	Frame           pageFrame
	Heading         headingView
	ConnectionsPath string
	CompanionPath   string
	SettingsPath    string
	Choice          fetchsettings.Choice
}

func (p *pages) settings(w http.ResponseWriter, r *http.Request) {
	rows := p.deviceRows()
	render(w, r, settingsPage, settingsView{
		Frame:           p.frame,
		Heading:         p.headingFor(rows, settingsTab),
		ConnectionsPath: pathConnections,
		CompanionPath:   Path,
		SettingsPath:    pathSettings,
		Choice:          p.fetchSettings.Choice(),
	})
}

func (p *pages) choose(w http.ResponseWriter, r *http.Request) {
	p.fetchSettings.Choose(r.Context(), p.choiceFrom(r))
	http.Redirect(w, r, pathSettings, http.StatusSeeOther)
}

func (p *pages) choiceFrom(r *http.Request) fetchsettings.Choice {
	choice := p.fetchSettings.Choice()
	if choice.CompanionsFetch {
		choice.PeerFillsIn = r.PostFormValue(fieldPeerFillsIn) != ""
	}
	choice.CompanionsFetch = r.PostFormValue(fieldCompanionsFetch) != ""
	return choice
}
