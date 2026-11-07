// Package companionspage serves the Companions pages of the web UI: the list of the paired
// devices, where the operator unpairs a device, the settings of who fetches pages, and the page
// that pairs a device with a QR code and shows how the pairing goes while it happens.
package companionspage

import (
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
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/fetchsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pairingoffers"
)

const (
	Path              = "/companions"
	pathUnpair        = Path + "/unpair"
	pathSettings      = Path + "/settings"
	pathConnections   = Path + "/connections"
	pathPair          = Path + "/pair"
	pathPairingEvents = pathPair + "/events"
	pathAssets        = Path + "/assets/"
	appGuideURL       = "https://github.com/nikitakarpei/my-yacy-node/tree/main/" +
		"distributions/yacypeercompanion"
	connectionTimeLayout = "2 Jan 2006, 15:04"
)

var (
	//go:embed templates
	templateFiles embed.FS
	//go:embed assets
	assets embed.FS

	companionsPage = template.Must(template.ParseFS(templateFiles,
		"templates/companions.html", "templates/heading.html"))
	settingsPage = template.Must(template.ParseFS(templateFiles,
		"templates/settings.html", "templates/heading.html"))
	pairPage = template.Must(template.ParseFS(templateFiles, "templates/pair.html"))
)

type PairedDevices interface {
	List() []paireddevices.Device
}

type Unpairing interface {
	Unpair(ctx context.Context, device string)
}

type PairingOffer interface {
	Events() <-chan pairingoffers.Event
	Close()
}

type PairingOffers interface {
	Open() PairingOffer
}

type Broker interface {
	ConnectedUsers() map[string]bool
}

type FetchSettings interface {
	Choice() fetchsettings.Choice
	Choose(ctx context.Context, choice fetchsettings.Choice)
}

type Collaborators struct {
	Devices   PairedDevices
	Unpairing Unpairing
	Offers    PairingOffers
	Broker    Broker
	Settings  FetchSettings
}

type pages struct {
	collaborators Collaborators
	link          companionprotocol.PairLink
	frame         pageFrame
}

type pageFrame struct {
	WebUIStylesheet string
	TabBar          template.HTML
	AppGuideURL     string
}

func Register(
	mux *http.ServeMux,
	collaborators Collaborators,
	link companionprotocol.PairLink,
	tabBar webuistyle.TabBar,
) {
	served := &pages{
		collaborators: collaborators,
		link:          link,
		frame: pageFrame{
			WebUIStylesheet: webuistyle.StylesheetPath,
			TabBar:          tabBar.HTML(),
			AppGuideURL:     appGuideURL,
		},
	}
	mux.HandleFunc("GET "+Path, served.companions)
	mux.HandleFunc("GET "+pathConnections, served.connections)
	mux.HandleFunc("POST "+pathUnpair, served.unpair)
	mux.HandleFunc("GET "+pathSettings, served.settings)
	mux.HandleFunc("POST "+pathSettings, served.choose)
	mux.HandleFunc("GET "+pathPair, served.pair)
	mux.HandleFunc("GET "+pathPairingEvents, served.pairingEvents)
	mux.Handle("GET "+pathAssets, http.StripPrefix(Path, http.FileServerFS(assets)))
}

type companionsView struct {
	Frame           pageFrame
	Heading         headingView
	ConnectionsPath string
	JustPaired      string
	Unpairing       *deviceRow
	Devices         []deviceRow
	PairPath        string
	UnpairPath      string
	CompanionPath   string
}

type deviceRow struct {
	ID            string `json:"device"`
	Name          string `json:"-"`
	AppVersion    string `json:"appVersion"`
	LastConnected string `json:"lastConnected"`
	Connected     bool   `json:"connected"`
}

func (p *pages) companions(w http.ResponseWriter, r *http.Request) {
	rows := p.deviceRows()
	view := companionsView{
		Frame:           p.frame,
		Heading:         p.headingFor(rows, devicesTab),
		ConnectionsPath: pathConnections,
		Devices:         rows,
		PairPath:        pathPair,
		UnpairPath:      pathUnpair,
		CompanionPath:   Path,
	}
	if paired := rowOf(rows, r.URL.Query().Get("paired")); paired != nil {
		view.JustPaired = paired.Name
	}
	view.Unpairing = rowOf(rows, r.URL.Query().Get("unpair"))
	render(w, r, companionsPage, view)
}

func (p *pages) deviceRows() []deviceRow {
	connectedUsers := p.collaborators.Broker.ConnectedUsers()
	devices := p.collaborators.Devices.List()
	slices.SortFunc(devices, func(a, b paireddevices.Device) int {
		return cmp.Compare(a.Name, b.Name)
	})
	rows := make([]deviceRow, 0, len(devices))
	for _, device := range devices {
		rows = append(rows, deviceRow{
			ID:            device.ID,
			Name:          device.Name,
			AppVersion:    device.AppVersion,
			LastConnected: connectionTimeOf(device.LastConnectedAt),
			Connected:     connectedUsers[companionaccess.UserOf(device.ID)],
		})
	}
	return rows
}

func (p *pages) connections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(p.deviceRows()); err != nil {
		slog.DebugContext(r.Context(), "companions page reader left", slog.Any("error", err))
	}
}

func connectionTimeOf(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.Local().Format(connectionTimeLayout)
}

func rowOf(rows []deviceRow, id string) *deviceRow {
	position := slices.IndexFunc(rows, func(row deviceRow) bool { return row.ID == id })
	if id == "" || position < 0 {
		return nil
	}
	return &rows[position]
}

func (p *pages) unpair(w http.ResponseWriter, r *http.Request) {
	device := r.PostFormValue("device")
	if device != "" {
		p.collaborators.Unpairing.Unpair(r.Context(), device)
	}
	http.Redirect(w, r, Path, http.StatusSeeOther)
}

func render(w http.ResponseWriter, r *http.Request, page *template.Template, view any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Execute(w, view); err != nil {
		slog.DebugContext(r.Context(), "companions page reader left", slog.Any("error", err))
	}
}
