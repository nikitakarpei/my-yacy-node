package egressproxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/netip"
)

const tunnelEstablished = "HTTP/1.1 200 Connection established\r\n\r\n"

type overlayFront struct {
	routes []overlayRoute
	beyond http.Handler
}

type overlayRoute struct {
	overlay   Overlay
	forwarder *httputil.ReverseProxy
}

func overlayFrontFor(overlays []Overlay, beyond http.Handler) overlayFront {
	routes := make([]overlayRoute, 0, len(overlays))
	for _, overlay := range overlays {
		routes = append(routes, overlayRoute{
			overlay: overlay,
			forwarder: &httputil.ReverseProxy{
				Rewrite:      func(*httputil.ProxyRequest) {},
				Transport:    &http.Transport{DialContext: overlay.Dialer.DialContext},
				ErrorHandler: reportForwardFailure,
			},
		})
	}
	return overlayFront{routes: routes, beyond: beyond}
}

func reportForwardFailure(w http.ResponseWriter, request *http.Request, err error) {
	slog.WarnContext(request.Context(), "overlay request failed",
		slog.String("destination", request.URL.Host),
		slog.Any("reason", err),
	)
	w.WriteHeader(http.StatusBadGateway)
}

func (f overlayFront) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	route, isOverlay := f.routeFor(request)
	if !isOverlay {
		f.beyond.ServeHTTP(w, request)
		return
	}
	slog.DebugContext(request.Context(), "overlay request received",
		slog.String("method", request.Method),
		slog.String("destination", request.URL.Host),
	)
	if request.Method == http.MethodConnect {
		route.tunnel(w, request)
		return
	}
	route.forwarder.ServeHTTP(w, request)
}

func (f overlayFront) routeFor(request *http.Request) (overlayRoute, bool) {
	destination, _ := netip.ParseAddr(request.URL.Hostname())
	for _, route := range f.routes {
		if route.overlay.Addresses.Contains(destination.Unmap()) {
			return route, true
		}
	}
	return overlayRoute{}, false
}

func (r overlayRoute) tunnel(w http.ResponseWriter, request *http.Request) {
	target, err := r.overlay.Dialer.DialContext(request.Context(), "tcp", request.URL.Host)
	if err != nil {
		reportForwardFailure(w, request, err)
		return
	}
	defer func() { _ = target.Close() }()
	client, buffered, err := http.NewResponseController(w).Hijack()
	if err != nil {
		reportForwardFailure(w, request, err)
		return
	}
	defer func() { _ = client.Close() }()
	_, _ = buffered.WriteString(tunnelEstablished)
	_ = buffered.Flush()
	directionEnded := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(target, buffered)
		directionEnded <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, target)
		directionEnded <- struct{}{}
	}()
	<-directionEnded
}
