package poisonhalt_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
)

type stubMsg struct {
	metadata *jetstream.MsgMetadata
	metaErr  error
}

func (m stubMsg) Metadata() (*jetstream.MsgMetadata, error) { return m.metadata, m.metaErr }
func (stubMsg) Data() []byte                                { return nil }
func (stubMsg) Headers() nats.Header                        { return nil }
func (stubMsg) Subject() string                             { return "yacy.crawl.page.markdown" }
func (stubMsg) Reply() string                               { return "" }
func (stubMsg) Ack() error                                  { return nil }
func (stubMsg) DoubleAck(context.Context) error             { return nil }
func (stubMsg) Nak() error                                  { return nil }
func (stubMsg) NakWithDelay(time.Duration) error            { return nil }
func (stubMsg) InProgress() error                           { return nil }
func (stubMsg) Term() error                                 { return nil }
func (stubMsg) TermWithReason(string) error                 { return nil }

func TestHaltReturnsPoisonSentinel(t *testing.T) {
	cause := errors.New("bad json")
	msg := stubMsg{metadata: &jetstream.MsgMetadata{
		Stream:   "YACY_CRAWL_PAGE_MARKDOWN",
		Sequence: jetstream.SequencePair{Stream: 7},
	}}

	err := poisonhalt.Halt(context.Background(), msg, cause)
	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("Halt error = %v, want poison sentinel", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("Halt error = %v, want wrapped cause", err)
	}
}

func TestHaltReturnsSentinelWhenMetadataUnavailable(t *testing.T) {
	msg := stubMsg{metaErr: errors.New("no metadata")}
	err := poisonhalt.Halt(context.Background(), msg, errors.New("bad json"))
	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("Halt error = %v, want poison sentinel", err)
	}
}
