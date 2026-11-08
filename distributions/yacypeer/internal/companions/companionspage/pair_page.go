package companionspage

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

type pairView struct {
	Frame             pageFrame
	PairingEventsPath string
	CompanionsPath    string
	AssetsPath        string
}

func (p *pages) pair(w http.ResponseWriter, r *http.Request) {
	render(w, r, pairPage, pairView{
		Frame:             p.frame,
		PairingEventsPath: pathPairingEvents,
		CompanionsPath:    Path,
		AssetsPath:        pathAssets,
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
	offer := p.offers.Open()
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
			if _, isPaired := event.(pairingoffers.DevicePaired); isPaired {
				return
			}
		}
	}
}

type pairingBegunMessage struct {
	DeviceName companionprotocol.DeviceName `json:"deviceName"`
}

type devicePairedMessage struct {
	DeviceName companionprotocol.DeviceName `json:"deviceName"`
	URL        string                       `json:"url"`
}

type deviceRefusedMessage struct {
	DeviceName companionprotocol.DeviceName `json:"deviceName"`
	Cause      string                       `json:"cause"`
}

func (p *pages) writeEvent(w io.Writer, event pairingoffers.Event) error {
	switch event := event.(type) {
	case pairingoffers.SecretRenewed:
		link := p.linkWithoutSecret
		link.Secret = event.Secret
		return writeServerEvent(w, "code", string(pairingCodeOf(link.String())))
	case pairingoffers.PairingBegun:
		return writeMessage(w, "begun", pairingBegunMessage{DeviceName: event.DeviceName})
	case pairingoffers.DevicePaired:
		return writeMessage(w, "paired", devicePairedMessage{
			DeviceName: event.DeviceName,
			URL:        Path + "?" + url.Values{parameterPaired: {string(event.Device)}}.Encode(),
		})
	case pairingoffers.DeviceRefused:
		return writeMessage(w, "refused", deviceRefusedMessage{
			DeviceName: event.DeviceName, Cause: causeOf(event.Reason),
		})
	}
	return nil
}

func causeOf(reason companionprotocol.RefusalReason) string {
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

func writeMessage(w io.Writer, name string, message any) error {
	data, err := json.Marshal(message)
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
