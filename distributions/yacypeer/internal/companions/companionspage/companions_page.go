// Package companionspage serves the Companions pages of the web UI: the list of the paired
// devices, where the operator unpairs a device, and the page that pairs a device with a QR code
// and shows how the pairing goes while it happens.
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
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

const (
	Path              = "/companions"
	pathUnpair        = Path + "/unpair"
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

	companionsPage = template.Must(template.ParseFS(templateFiles, "templates/companions.html"))
	pairPage       = template.Must(template.ParseFS(templateFiles, "templates/pair.html"))
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

type Broker interface {
	ConnectedUsers() map[string]bool
}

type Collaborators struct {
	Devices  PairedDevices
	Unpairer Unpairer
	Offers   PairingOffers
	Broker   Broker
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
	mux.HandleFunc("GET "+pathPair, served.pair)
	mux.HandleFunc("GET "+pathPairingEvents, served.pairingEvents)
	mux.Handle("GET "+pathAssets, http.StripPrefix(Path, http.FileServerFS(assets)))
}

type companionsView struct {
	Frame           pageFrame
	ConnectionsPath string
	JustPaired      companionprotocol.DeviceName
	DeviceToUnpair  *deviceRow
	Devices         []deviceRow
	PairPath        string
	UnpairPath      string
	CompanionPath   string
}

type deviceRow struct {
	ID            companionprotocol.DeviceID   `json:"device"`
	Name          companionprotocol.DeviceName `json:"-"`
	AppVersion    string                       `json:"appVersion"`
	LastConnected string                       `json:"lastConnected"`
	Connected     bool                         `json:"connected"`
}

func (p *pages) companions(w http.ResponseWriter, r *http.Request) {
	rows := p.deviceRows()
	view := companionsView{
		Frame:           p.frame,
		ConnectionsPath: pathConnections,
		Devices:         rows,
		PairPath:        pathPair,
		UnpairPath:      pathUnpair,
		CompanionPath:   Path,
	}
	if paired := rowOf(
		rows,
		companionprotocol.DeviceID(r.URL.Query().Get("paired")),
	); paired != nil {
		view.JustPaired = paired.Name
	}
	view.DeviceToUnpair = rowOf(rows, companionprotocol.DeviceID(r.URL.Query().Get("unpair")))
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
	device := companionprotocol.DeviceID(r.PostFormValue("device"))
	if device != "" {
		p.collaborators.Unpairer.Unpair(r.Context(), device)
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
