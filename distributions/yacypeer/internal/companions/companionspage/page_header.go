package companionspage

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

type tab int

const (
	devicesTab tab = iota
	settingsTab
)

type headerView struct {
	CompanionsPath      string
	SettingsPath        string
	SettingsTabCurrent  bool
	OnlyCompanionsFetch bool
	Paused              bool
	PairedDeviceName    companionprotocol.DeviceName
}

func (p *pages) headerFor(rows []deviceRow, current tab) headerView {
	choice := p.companionSettings.Choice()
	onlyCompanionsFetch := len(rows) > 0 && choice.OnlyCompanionsFetch()
	return headerView{
		CompanionsPath:      Path,
		SettingsPath:        pathSettings,
		SettingsTabCurrent:  current == settingsTab,
		OnlyCompanionsFetch: onlyCompanionsFetch,
		Paused:              onlyCompanionsFetch && !slices.ContainsFunc(rows, isConnected),
	}
}

func isConnected(row deviceRow) bool {
	return row.Connected
}
