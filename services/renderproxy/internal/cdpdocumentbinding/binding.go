// Package cdpdocumentbinding binds the main document request from a stream of CDP network
// events and records the status, content type, and response header observed for it.
package cdpdocumentbinding

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/network"
)

const msgHeaderFieldNotText = "document response header field is not text"

type Binding struct {
	ctx            context.Context
	mu             sync.Mutex
	requestID      network.RequestID
	bound          bool
	statusCode     int
	contentType    string
	responseHeader http.Header
	seen           bool
}

func New(ctx context.Context) *Binding {
	return &Binding{ctx: ctx}
}

func (m *Binding) Observe(event any) {
	switch e := event.(type) {
	case *network.EventRequestWillBeSent:
		if e.Type != network.ResourceTypeDocument {
			return
		}
		m.mu.Lock()
		if !m.bound {
			m.requestID = e.RequestID
			m.bound = true
		}
		m.mu.Unlock()
	case *network.EventResponseReceived:
		if e.Response == nil {
			return
		}
		m.mu.Lock()
		if m.bound && e.RequestID == m.requestID {
			m.statusCode = int(e.Response.Status)
			m.contentType = e.Response.MimeType
			m.responseHeader = headerOf(m.ctx, e.Response.Headers)
			m.seen = true
		}
		m.mu.Unlock()
	}
}

func headerOf(ctx context.Context, networkHeaders network.Headers) http.Header {
	header := http.Header{}
	for name, field := range networkHeaders {
		text, isText := field.(string)
		if !isText {
			slog.WarnContext(ctx, msgHeaderFieldNotText, slog.String("headerName", name))
			continue
		}
		for _, value := range strings.Split(text, "\n") {
			header.Add(name, value)
		}
	}
	return header
}

type BoundDocument struct {
	StatusCode     int
	ContentType    string
	ResponseHeader http.Header
	Seen           bool
}

func (m *Binding) BoundDocument() BoundDocument {
	m.mu.Lock()
	defer m.mu.Unlock()
	return BoundDocument{
		StatusCode:     m.statusCode,
		ContentType:    m.contentType,
		ResponseHeader: m.responseHeader,
		Seen:           m.seen,
	}
}
