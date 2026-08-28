// Package pullintaketest delivers messages to a pull intake without a broker,
// and records how the intake settled each one.
package pullintaketest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	Subject = "subj"

	Acknowledged = "ack"
	Returned     = "nak"
	HeldBack     = "nak-with-delay"
	Terminated   = "term"
)

type Message struct {
	Body          []byte
	Stream        string
	Sequence      uint64
	MetadataError error

	mu          sync.Mutex
	settlements []string
	delay       time.Duration
}

func (m *Message) Subject() string      { return Subject }
func (m *Message) Reply() string        { return "" }
func (m *Message) Data() []byte         { return m.Body }
func (m *Message) Headers() nats.Header { return nil }
func (m *Message) InProgress() error    { return nil }

func (m *Message) Ack() error                      { return m.settle(Acknowledged) }
func (m *Message) DoubleAck(context.Context) error { return m.settle(Acknowledged) }
func (m *Message) Nak() error                      { return m.settle(Returned) }
func (m *Message) Term() error                     { return m.settle(Terminated) }
func (m *Message) TermWithReason(string) error     { return m.settle(Terminated) }

func (m *Message) NakWithDelay(delay time.Duration) error {
	m.mu.Lock()
	m.delay = delay
	m.mu.Unlock()
	return m.settle(HeldBack)
}

func (m *Message) Metadata() (*jetstream.MsgMetadata, error) {
	if m.MetadataError != nil {
		return nil, m.MetadataError
	}
	return &jetstream.MsgMetadata{
		Stream:   m.Stream,
		Sequence: jetstream.SequencePair{Stream: m.Sequence},
	}, nil
}

func (m *Message) settle(action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settlements = append(m.settlements, action)
	return nil
}

func (m *Message) Settlements() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.settlements...)
}

func (m *Message) Settlement(t *testing.T) string {
	t.Helper()
	settled := m.Settlements()
	if len(settled) != 1 {
		t.Fatalf("message settled %v, want exactly one action", settled)
	}
	return settled[0]
}

func (m *Message) HeldBackFor() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.delay
}
