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
	pathDeviceStates  = Path + "/device-states"
	pathPair          = Path + "/pair"
	pathPairingEvents = pathPair + "/events"
	pathAssets        = Path + "/assets/"
	parameterPaired   = "paired"
	parameterUnpair   = "unpair"
	fieldDevice       = "device"
	appGuideURL       = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"distributions/yacypeercompanion/README.md"
	connectionTimeLayout = "2 Jan 2006, 15:04"
)

var (
	//go:embed templates
	templateFiles embed.FS
	//go:embed assets
	assets embed.FS

	companionsPage = template.Must(template.ParseFS(templateFiles,
		"templates/companions.html", "templates/header.html"))
	settingsPage = template.Must(template.ParseFS(templateFiles,
		"templates/settings.html", "templates/header.html"))
	pairPage = template.Must(template.ParseFS(templateFiles, "templates/pair.html"))
)

type PairedDevices interface {
	List() []paireddevices.Device
}

type Unpairer interface {
	Unpair(ctx context.Context, device companionprotocol.DeviceID)
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
	Choice() companionsettings.Choice
	Choose(ctx context.Context, choice companionsettings.Choice)
}

type Collaborators struct {
	Devices     PairedDevices
	Unpairer    Unpairer
	Offers      PairingOffers
	Connections DeviceConnections
	Settings    CompanionSettings
}

type pages struct {
	devices           PairedDevices
	unpairer          Unpairer
	offers            PairingOffers
	deviceConnections DeviceConnections
	companionSettings CompanionSettings
	linkWithoutSecret companionprotocol.PairLink
	frame             pageFrame
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
	companionPages := &pages{
		devices:           collaborators.Devices,
		unpairer:          collaborators.Unpairer,
		offers:            collaborators.Offers,
		deviceConnections: collaborators.Connections,
		companionSettings: collaborators.Settings,
		linkWithoutSecret: linkWithoutSecret,
		frame: pageFrame{
			WebUIStylesheet: webuistyle.StylesheetPath,
			TabBar:          tabBar.HTML(),
			AppGuideURL:     appGuideURL,
		},
	}
	mux.HandleFunc("GET "+Path, companionPages.companions)
	mux.HandleFunc("GET "+pathDeviceStates, companionPages.deviceStates)
	mux.HandleFunc("POST "+pathUnpair, companionPages.unpair)
	mux.HandleFunc("GET "+pathSettings, companionPages.settings)
	mux.HandleFunc("POST "+pathSettings, companionPages.choose)
	mux.HandleFunc("GET "+pathPair, companionPages.pair)
	mux.HandleFunc("GET "+pathPairingEvents, companionPages.pairingEvents)
	mux.Handle("GET "+pathAssets, http.StripPrefix(Path, http.FileServerFS(assets)))
}

type companionsView struct {
	Frame            pageFrame
	Header           headerView
	DeviceStatesPath string
	DeviceToUnpair   *deviceRow
	Devices          []deviceRow
	PairPath         string
	UnpairPath       string
	CompanionsPath   string
	AssetsPath       string
	PairedParameter  string
	UnpairParameter  string
	DeviceField      string
}

type deviceRow struct {
	ID                 companionprotocol.DeviceID   `json:"device"`
	Name               companionprotocol.DeviceName `json:"-"`
	AppVersion         string                       `json:"appVersion"`
	LastConnectionTime string                       `json:"lastConnectionTime"`
	Connected          bool                         `json:"connected"`
}

func (p *pages) companions(w http.ResponseWriter, r *http.Request) {
	rows := p.deviceRows()
	view := companionsView{
		Frame:            p.frame,
		Header:           p.headerFor(rows, devicesTab),
		DeviceStatesPath: pathDeviceStates,
		Devices:          rows,
		PairPath:         pathPair,
		UnpairPath:       pathUnpair,
		CompanionsPath:   Path,
		AssetsPath:       pathAssets,
		PairedParameter:  parameterPaired,
		UnpairParameter:  parameterUnpair,
		DeviceField:      fieldDevice,
	}
	if pairedRow := rowOf(
		rows,
		companionprotocol.DeviceID(r.URL.Query().Get(parameterPaired)),
	); pairedRow != nil {
		view.Header.PairedDeviceName = pairedRow.Name
	}
	view.DeviceToUnpair = rowOf(
		rows,
		companionprotocol.DeviceID(r.URL.Query().Get(parameterUnpair)),
	)
	render(w, r, companionsPage, view)
}

func (p *pages) deviceRows() []deviceRow {
	connectedDevices := p.deviceConnections.ConnectedDevices()
	devices := p.devices.List()
	slices.SortFunc(devices, func(a, b paireddevices.Device) int {
		return cmp.Compare(a.Name, b.Name)
	})
	rows := make([]deviceRow, 0, len(devices))
	for _, device := range devices {
		rows = append(rows, deviceRow{
			ID:                 device.ID,
			Name:               device.Name,
			AppVersion:         device.AppVersion,
			LastConnectionTime: connectionTimeOf(device.LastConnectedAt),
			Connected:          connectedDevices[device.ID],
		})
	}
	return rows
}

func (p *pages) deviceStates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	encoded, err := json.Marshal(p.deviceRows())
	if err != nil {
		slog.ErrorContext(r.Context(), "device connections not encoded", slog.Any("error", err))
		http.Error(w, "device connections not encoded", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(encoded); err != nil {
		slog.DebugContext(r.Context(), "companions page reader left", slog.Any("error", err))
	}
}

func connectionTimeOf(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.Local().Format(connectionTimeLayout)
}

func rowOf(rows []deviceRow, id companionprotocol.DeviceID) *deviceRow {
	position := slices.IndexFunc(rows, func(row deviceRow) bool { return row.ID == id })
	if id == "" || position < 0 {
		return nil
	}
	return &rows[position]
}

func (p *pages) unpair(w http.ResponseWriter, r *http.Request) {
	deviceID := companionprotocol.DeviceID(r.PostFormValue(fieldDevice))
	if deviceID != "" {
		p.unpairer.Unpair(r.Context(), deviceID)
	}
	http.Redirect(w, r, Path, http.StatusSeeOther)
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
