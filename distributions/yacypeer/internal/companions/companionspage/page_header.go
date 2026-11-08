package companionspage

import "github.com/nikitakarpei/yacy-rwi-node/companionprotocol"

type tab int

const (
	devicesTab tab = iota
	settingsTab
)

type headerView struct {
	DevicesPath        string
	SettingsPath       string
	DevicesTabCurrent  bool
	SettingsTabCurrent bool
	FetchingPaused     bool
	PairedDeviceName   companionprotocol.DeviceName
}

func headerFor(fetchingPaused bool, current tab) headerView {
	return headerView{
		DevicesPath:        Path,
		SettingsPath:       pathSettings,
		DevicesTabCurrent:  current == devicesTab,
		SettingsTabCurrent: current == settingsTab,
		FetchingPaused:     fetchingPaused,
	}
}
