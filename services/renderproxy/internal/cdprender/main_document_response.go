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

type mainDocumentResult struct {
	statusCode  int
	contentType string
	requestID   network.RequestID
	seen        bool
}

func (m *mainDocumentResponse) result() mainDocumentResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	return mainDocumentResult{
		statusCode:  m.statusCode,
		contentType: m.contentType,
		requestID:   m.requestID,
		seen:        m.seen,
	}
}
