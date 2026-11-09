package companionspage

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
)

type PairPage struct {
	offers            PairingOffers
	linkWithoutSecret companionprotocol.PairLink
	frame             pageFrame
}

func NewPairPage(
	offers PairingOffers,
	linkWithoutSecret companionprotocol.PairLink,
	tabBar webuistyle.TabBar,
) *PairPage {
	return &PairPage{
		offers:            offers,
		linkWithoutSecret: linkWithoutSecret,
		frame:             frameWith(tabBar),
	}
}

type pairView struct {
	Frame             pageFrame
	PairingEventsPath string
	CompanionsPath    string
	AssetsPath        string
}

func (p *PairPage) show(w http.ResponseWriter, r *http.Request) {
	render(w, r, pairTemplate, pairView{
		Frame:             p.frame,
		PairingEventsPath: pathPairingEvents,
		CompanionsPath:    Path,
		AssetsPath:        pathAssets,
	})
}

func (p *PairPage) pairingEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	stream := http.NewResponseController(w)
	if err := stream.Flush(); err != nil {
		slog.DebugContext(r.Context(), "pairing page left", slog.Any("error", err))
		return
	}
	offer := p.offers.Open()
	defer offer.Close()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-offer.Events():
			err := p.writeEvent(w, event)
			if err == nil {
				err = stream.Flush()
			}
			if err != nil {
				slog.DebugContext(r.Context(), "pairing page left", slog.Any("error", err))
				return
			}
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
	DeviceName    companionprotocol.DeviceName `json:"deviceName"`
	CompanionsURL string                       `json:"companionsURL"`
}

type deviceRefusedMessage struct {
	DeviceName companionprotocol.DeviceName `json:"deviceName"`
	Reason     string                       `json:"reason"`
}

func (p *PairPage) writeEvent(w io.Writer, event pairingoffers.Event) error {
	switch event := event.(type) {
	case pairingoffers.SecretRenewed:
		link := p.linkWithoutSecret
		link.Secret = event.Secret
		return writeServerEvent(w, "code", string(pairingCodeFrom(link.String())))
	case pairingoffers.PairingBegun:
		return writeMessage(w, "begun", pairingBegunMessage{DeviceName: event.DeviceName})
	case pairingoffers.DevicePaired:
		return writeMessage(w, "paired", devicePairedMessage{
			DeviceName: event.DeviceName,
			CompanionsURL: Path + "?" + url.Values{
				parameterPaired: {string(event.DeviceID)},
			}.Encode(),
		})
	case pairingoffers.DeviceRefused:
		return writeMessage(w, "refused", deviceRefusedMessage{
			DeviceName: event.DeviceName, Reason: reasonTextOf(event.Reason),
		})
	}
	return nil
}

func reasonTextOf(reason companionprotocol.RefusalReason) string {
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
	data, _ := json.Marshal(message) //nolint:errchkjson // only strings.
	return writeServerEvent(w, name, string(data))
}

func writeServerEvent(w io.Writer, name string, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return fmt.Errorf("write %s event: %w", name, err)
	}
	return nil
}
