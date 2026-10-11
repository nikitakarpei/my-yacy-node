package pullintaketest

import (
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type MessageSource struct {
	queue *messageQueue
}

func MessageSourceOf(messages ...jetstream.Msg) MessageSource {
	return MessageSource{
		queue: &messageQueue{messages: messages, endError: nats.ErrConnectionClosed},
	}
}

func MessageSourceThatFails(failure error) MessageSource {
	return MessageSource{queue: &messageQueue{endError: failure}}
}

func MessageSourceHeldOnItsFirstMessage(
	release <-chan struct{},
	messages ...jetstream.Msg,
) MessageSource {
	return MessageSource{queue: &messageQueue{
		messages:  messages,
		heldUntil: release,
		endError:  nats.ErrConnectionClosed,
	}}
}

func (s MessageSource) Next(...jetstream.FetchOpt) (jetstream.Msg, error) {
	message, held := s.queue.takeOne()
	if message == nil {
		return nil, s.queue.endError
	}
	if held != nil {
		<-held
	}
	return message, nil
}

func (s MessageSource) AmountOfTakenMessages() int {
	s.queue.mu.Lock()
	defer s.queue.mu.Unlock()
	return s.queue.taken
}

type messageQueue struct {
	mu        sync.Mutex
	messages  []jetstream.Msg
	taken     int
	heldUntil <-chan struct{}
	endError  error
}

func (q *messageQueue) takeOne() (jetstream.Msg, <-chan struct{}) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.messages) == 0 {
		return nil, nil
	}
	message := q.messages[0]
	q.messages = q.messages[1:]
	q.taken++
	held := q.heldUntil
	q.heldUntil = nil
	return message, held
}
