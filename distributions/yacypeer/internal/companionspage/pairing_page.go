package companionspage

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pairingoffers"
)

type pairView struct {
	Frame         pageFrame
	EventsPath    string
	CompanionPath string
}

func (p *pages) pair(w http.ResponseWriter, r *http.Request) {
	render(w, r, pairPage, pairView{
		Frame:         p.frame,
		EventsPath:    pathPairingEvents,
		CompanionPath: Path,
	})
}

func (p *pages) pairingEvents(w http.ResponseWriter, r *http.Request) {
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	flusher.Flush()
	offer := p.collaborators.Offers.Open()
	defer offer.Close()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-offer.Events():
			if err := p.writeEvent(w, event); err != nil {
				slog.DebugContext(r.Context(), "pairing page left", slog.Any("error", err))
				return
			}
			flusher.Flush()
			if event.Kind == pairingoffers.DevicePaired {
				return
			}
		}
	}
}

type deviceEvent struct {
	DeviceName string `json:"deviceName"`
	Cause      string `json:"cause,omitempty"`
	URL        string `json:"url,omitempty"`
}

func (p *pages) writeEvent(w io.Writer, event pairingoffers.Event) error {
	switch event.Kind {
	case pairingoffers.SecretRenewed:
		link := p.link
		link.Secret = event.Secret
		return writeServerEvent(w, "code", string(pairingCodeOf(link.String())))
	case pairingoffers.PairingBegun:
		return writeDeviceEvent(w, "started", deviceEvent{DeviceName: event.DeviceName})
	case pairingoffers.DevicePaired:
		return writeDeviceEvent(w, "paired", deviceEvent{
			DeviceName: event.DeviceName,
			URL:        Path + "?" + url.Values{"paired": {event.Device}}.Encode(),
		})
	case pairingoffers.DeviceRefused:
		return writeDeviceEvent(w, "refused", deviceEvent{
			DeviceName: event.DeviceName, Cause: causeOf(event.Reason),
		})
	}
	return nil
}

func causeOf(reason companionprotocol.Reason) string {
	switch reason {
	case companionprotocol.ReasonAppOutdated:
		return "The app on the device is outdated. Please update the app."
	case companionprotocol.ReasonPeerOutdated:
		return "The app on the device requires a newer YaCy peer. Please update the YaCy peer."
	case companionprotocol.ReasonAppNotOfficial:
		return "The app on the device is not an official build."
	case companionprotocol.ReasonDeviceNotTrusted:
		return "The device is not trusted."
	case companionprotocol.ReasonRequestInvalid:
		return "The device sent an invalid pairing request."
	}
	return ""
}

func writeDeviceEvent(w io.Writer, name string, event deviceEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", name, err)
	}
	return writeServerEvent(w, name, string(data))
}

func writeServerEvent(w io.Writer, name string, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return fmt.Errorf("write %s event: %w", name, err)
	}
	return nil
}
