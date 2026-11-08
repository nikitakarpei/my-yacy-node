// Package companionspage serves the Companions pages of the web UI: the list of the paired
// devices, where the operator unpairs a device, the settings of who fetches pages, and the page
// that pairs a device with a QR code and shows how the pairing goes while it happens.
package companionspage

import (
	"bytes"
	"cmp"
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

const (
	Path              = "/companions"
	pathUnpair        = Path + "/unpair"
	pathSettings      = Path + "/settings"
	pathStates        = Path + "/states"
	pathPair          = Path + "/pair"
	pathPairingEvents = pathPair + "/events"
	pathAssets        = Path + "/assets/"
	parameterPaired   = "paired"
	parameterUnpair   = "unpair"
	fieldDevice       = "device"
	appGuideURL       = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"distributions/yacypeercompanion/README.md"
	lastConnectedLayout = "2 Jan 2006, 15:04"
)

var (
	//go:embed templates
	templateFiles embed.FS
	//go:embed assets
	assets embed.FS

	devicesTemplate = template.Must(template.ParseFS(templateFiles,
		"templates/companions.html", "templates/header.html"))
	settingsTemplate = template.Must(template.ParseFS(templateFiles,
		"templates/settings.html", "templates/header.html"))
	pairTemplate = template.Must(template.ParseFS(templateFiles, "templates/pair.html"))
)

type PairedDevices interface {
	List() []paireddevices.Device
}

type Unpairer interface {
	Unpair(ctx context.Context, deviceID companionprotocol.DeviceID)
}

type PairingOffer interface {
	Events() <-chan pairingoffers.Event
	Close()
}

type PairingOffers interface {
	Open() PairingOffer
}

type DeviceConnections interface {
	ConnectedDevices() map[companionprotocol.DeviceID]bool
}

type CompanionSettings interface {
	Fetching() companionsettings.Fetching
	ChangeFetching(ctx context.Context, fetching companionsettings.Fetching)
}

type FetchingPause interface {
	FetchingPaused() bool
}

type Collaborators struct {
	Devices           PairedDevices
	Unpairer          Unpairer
	Offers            PairingOffers
	DeviceConnections DeviceConnections
	Settings          CompanionSettings
	FetchingPause     FetchingPause
}

type pageFrame struct {
	WebUIStylesheet string
	TabBar          template.HTML
	AppGuideURL     string
}

func Register(
	mux *http.ServeMux,
	collaborators Collaborators,
	linkWithoutSecret companionprotocol.PairLink,
	tabBar webuistyle.TabBar,
) {
	frame := pageFrame{
		WebUIStylesheet: webuistyle.StylesheetPath,
		TabBar:          tabBar.HTML(),
		AppGuideURL:     appGuideURL,
	}
	devices := &devicesPage{
		devices:           collaborators.Devices,
		unpairer:          collaborators.Unpairer,
		deviceConnections: collaborators.DeviceConnections,
		fetchingPause:     collaborators.FetchingPause,
		frame:             frame,
	}
	settings := &settingsPage{
		settings:      collaborators.Settings,
		fetchingPause: collaborators.FetchingPause,
		frame:         frame,
	}
	pair := &pairPage{
		offers:            collaborators.Offers,
		linkWithoutSecret: linkWithoutSecret,
		frame:             frame,
	}
	mux.HandleFunc("GET "+Path, devices.show)
	mux.HandleFunc("GET "+pathStates, devices.states)
	mux.HandleFunc("POST "+pathUnpair, devices.unpair)
	mux.HandleFunc("GET "+pathSettings, settings.show)
	mux.HandleFunc("POST "+pathSettings, settings.change)
	mux.HandleFunc("GET "+pathPair, pair.show)
	mux.HandleFunc("GET "+pathPairingEvents, pair.pairingEvents)
	mux.Handle("GET "+pathAssets, http.StripPrefix(Path, http.FileServerFS(assets)))
}

type devicesPage struct {
	devices           PairedDevices
	unpairer          Unpairer
	deviceConnections DeviceConnections
	fetchingPause     FetchingPause
	frame             pageFrame
}

type devicesView struct {
	Frame           pageFrame
	Header          headerView
	StatesPath      string
	DeviceToUnpair  *deviceRow
	Devices         deviceRows
	PairPath        string
	UnpairPath      string
	CompanionsPath  string
	AssetsPath      string
	PairedParameter string
	UnpairParameter string
	DeviceField     string
}

type deviceRow struct {
	ID            companionprotocol.DeviceID   `json:"id"`
	Name          companionprotocol.DeviceName `json:"-"`
	AppVersion    string                       `json:"appVersion"`
	LastConnected string                       `json:"lastConnected"`
	Connected     bool                         `json:"connected"`
}

type deviceRows []deviceRow

func (d *devicesPage) show(w http.ResponseWriter, r *http.Request) {
	rows := d.deviceRows()
	view := devicesView{
		Frame:           d.frame,
		Header:          headerFor(d.fetchingPause.FetchingPaused(), devicesTab),
		StatesPath:      pathStates,
		Devices:         rows,
		PairPath:        pathPair,
		UnpairPath:      pathUnpair,
		CompanionsPath:  Path,
		AssetsPath:      pathAssets,
		PairedParameter: parameterPaired,
		UnpairParameter: parameterUnpair,
		DeviceField:     fieldDevice,
	}
	if pairedRow := rows.with(
		companionprotocol.DeviceID(r.URL.Query().Get(parameterPaired)),
	); pairedRow != nil {
		view.Header.PairedDeviceName = pairedRow.Name
	}
	view.DeviceToUnpair = rows.with(companionprotocol.DeviceID(r.URL.Query().Get(parameterUnpair)))
	render(w, r, devicesTemplate, view)
}

func (d *devicesPage) deviceRows() deviceRows {
	connectedDevices := d.deviceConnections.ConnectedDevices()
	devices := d.devices.List()
	slices.SortFunc(devices, func(a, b paireddevices.Device) int {
		return cmp.Compare(a.Name, b.Name)
	})
	rows := make(deviceRows, 0, len(devices))
	for _, device := range devices {
		rows = append(rows, deviceRow{
			ID:            device.ID,
			Name:          device.Name,
			AppVersion:    device.AppVersion,
			LastConnected: lastConnectedFrom(device.LastConnectedAt),
			Connected:     connectedDevices[device.ID],
		})
	}
	return rows
}

func lastConnectedFrom(lastConnectedAt time.Time) string {
	if lastConnectedAt.IsZero() {
		return ""
	}
	return lastConnectedAt.Local().Format(lastConnectedLayout)
}

func (rows deviceRows) with(deviceID companionprotocol.DeviceID) *deviceRow {
	position := slices.IndexFunc(rows, func(row deviceRow) bool { return row.ID == deviceID })
	if position < 0 {
		return nil
	}
	return &rows[position]
}

func render(w http.ResponseWriter, r *http.Request, page *template.Template, view any) {
	var rendered bytes.Buffer
	if err := page.Execute(&rendered, view); err != nil {
		slog.ErrorContext(r.Context(), "companions page not rendered",
			slog.String("page", page.Name()), slog.Any("error", err),
		)
		http.Error(w, "page not rendered", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := rendered.WriteTo(w); err != nil {
		slog.DebugContext(r.Context(), "companions page reader left", slog.Any("error", err))
	}
}

type pageStates struct {
	Devices        deviceRows `json:"devices"`
	FetchingPaused bool       `json:"fetchingPaused"`
}

func (d *devicesPage) states(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(pageStates{
		Devices:        d.deviceRows(),
		FetchingPaused: d.fetchingPause.FetchingPaused(),
	}); err != nil {
		slog.DebugContext(r.Context(), "companions page states reader left", slog.Any("error", err))
	}
}

func (d *devicesPage) unpair(w http.ResponseWriter, r *http.Request) {
	d.unpairer.Unpair(r.Context(), companionprotocol.DeviceID(r.PostFormValue(fieldDevice)))
	http.Redirect(w, r, Path, http.StatusSeeOther)
}
