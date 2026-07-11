package cdprender

import (
	"sync"

	"github.com/chromedp/cdproto/network"
)

type mainDocumentResponse struct {
	mu          sync.Mutex
	requestID   network.RequestID
	bound       bool
	statusCode  int
	contentType string
	seen        bool
}

func (m *mainDocumentResponse) observe(event any) {
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
			m.seen = true
		}
		m.mu.Unlock()
	}
}

func (m *mainDocumentResponse) result() (statusCode int, contentType string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusCode, m.contentType, m.seen
}
