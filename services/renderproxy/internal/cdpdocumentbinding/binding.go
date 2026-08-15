// Package cdpdocumentbinding binds the main document request from a stream
// of CDP network events and records the status and content type observed for it.
package cdpdocumentbinding

import (
	"sync"

	"github.com/chromedp/cdproto/network"
)

type Binding struct {
	mu          sync.Mutex
	requestID   network.RequestID
	bound       bool
	statusCode  int
	contentType string
	seen        bool
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
			m.seen = true
		}
		m.mu.Unlock()
	}
}

type BoundDocument struct {
	StatusCode  int
	ContentType string
	Seen        bool
}

func (m *Binding) BoundDocument() BoundDocument {
	m.mu.Lock()
	defer m.mu.Unlock()
	return BoundDocument{
		StatusCode:  m.statusCode,
		ContentType: m.contentType,
		Seen:        m.seen,
	}
}
