package pullintaketest

import (
	"sync"

	"github.com/nats-io/nats.go/jetstream"
)

type MessageSource struct {
	iterator  *messageIterator
	openError error
}

func MessageSourceOf(messages ...jetstream.Msg) MessageSource {
	return MessageSource{iterator: &messageIterator{messages: messages}}
}

func MessageSourceThatCannotOpen(openError error) MessageSource {
	return MessageSource{openError: openError}
}

func MessageSourceHeldOnItsFirstMessage(
	release <-chan struct{},
	messages ...jetstream.Msg,
) MessageSource {
	return MessageSource{iterator: &messageIterator{
		messages:  messages,
		heldUntil: release,
	}}
}

func (s MessageSource) Messages(
	...jetstream.PullMessagesOpt,
) (jetstream.MessagesContext, error) {
	if s.openError != nil {
		return nil, s.openError
	}
	return s.iterator, nil
}

type messageIterator struct {
	mu        sync.Mutex
	messages  []jetstream.Msg
	heldUntil <-chan struct{}
}

func (it *messageIterator) Next(...jetstream.NextOpt) (jetstream.Msg, error) {
	message, held := it.takeOne()
	if message == nil {
		return nil, jetstream.ErrMsgIteratorClosed
	}
	if held != nil {
		<-held
	}
	return message, nil
}

func (it *messageIterator) takeOne() (jetstream.Msg, <-chan struct{}) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if len(it.messages) == 0 {
		return nil, nil
	}
	message := it.messages[0]
	it.messages = it.messages[1:]
	held := it.heldUntil
	it.heldUntil = nil
	return message, held
}

func (it *messageIterator) Stop()  {}
func (it *messageIterator) Drain() {}
