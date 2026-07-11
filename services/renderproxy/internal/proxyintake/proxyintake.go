// Package proxyintake accepts absolute-URL forward-proxy requests and refuses tunnels.
package proxyintake

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

const headerContentType = "Content-Type"

const (
	msgRenderFailed  = "render failed"
	msgRenderTimeout = "render timed out"
	msgWriteFailed   = "write rendered page to client failed"
)

type Handler struct {
	renderer renderedpage.Renderer
}

func New(renderer renderedpage.Renderer) *Handler {
	return &Handler{renderer: renderer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !r.URL.IsAbs() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	page, err := h.renderer.Render(r.Context(), r.URL.String())
	if err != nil {
		h.writeFailure(w, r.Context(), err)
		return
	}

	if page.ContentType != "" {
		w.Header().Set(headerContentType, page.ContentType)
	}
	w.WriteHeader(page.StatusCode)
	if _, err := w.Write(page.Body); err != nil {
		slog.WarnContext(r.Context(), msgWriteFailed, slog.Any("error", err))
	}
}

func (h *Handler) writeFailure(w http.ResponseWriter, ctx context.Context, err error) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		slog.WarnContext(ctx, msgRenderTimeout, slog.Any("error", err))
		w.WriteHeader(http.StatusGatewayTimeout)
		return
	}
	slog.WarnContext(ctx, msgRenderFailed, slog.Any("error", err))
	w.WriteHeader(http.StatusBadGateway)
}
