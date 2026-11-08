package companionspage

import "slices"

type tab int

const (
	devicesTab tab = iota
	settingsTab
)

type headingView struct {
	DevicesPath  string
	SettingsPath string
	SettingsTab  bool
	PauseRule    bool
	Paused       bool
	JustPaired   companionprotocol.DeviceName
}

func (p *pages) headingFor(rows []deviceRow, current tab) headingView {
	choice := p.collaborators.Settings.Choice()
	pauseRule := len(rows) > 0 && choice.CompanionsFetch && !choice.PeerFillsIn
	return headingView{
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
