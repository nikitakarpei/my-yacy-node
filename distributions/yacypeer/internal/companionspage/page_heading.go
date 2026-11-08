package companionspage

import "slices"

type tab int

const (
	devicesTab tab = iota
	settingsTab
)

type headingView struct {
	AppGuideURL  string
	PairPath     string
	DevicesPath  string
	SettingsPath string
	SettingsTab  bool
	PauseRule    bool
	Paused       bool
	JustPaired   string
}

func (p *pages) headingFor(rows []deviceRow, current tab) headingView {
	choice := p.collaborators.Settings.Choice()
	pauseRule := len(rows) > 0 && choice.CompanionsFetch && !choice.PeerFillsIn
	return headingView{
		AppGuideURL:  p.frame.AppGuideURL,
		PairPath:     pathPair,
		DevicesPath:  Path,
		SettingsPath: pathSettings,
		SettingsTab:  current == settingsTab,
		PauseRule:    pauseRule,
		Paused:       pauseRule && !slices.ContainsFunc(rows, isConnected),
	}
}

func isConnected(row deviceRow) bool {
	return row.Connected
}
