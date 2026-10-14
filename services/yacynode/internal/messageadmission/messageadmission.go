// Package messageadmission answers the messages that the operator of one peer
// writes to the operator of this node. It admits a message only from the peer
// the message names, within the announced size and an allowance per sender
// address, and gives it to the operator's inbox. Without an inbox it refuses
// every message.
package messageadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type Inbox interface {
	Deliver(ctx context.Context, message Message) bool
}

type Message struct {
	Subject string
	Body    string
	Sender  Sender
}

type Sender struct {
	Hash    yacymodel.Hash
	Name    string
	Address string
}

func MountAccepting(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	now func() time.Time,
	inbox Inbox,
) {
	httpguard.MountFields(
		router,
		yacyproto.PathMessage,
		yacyproto.MessageEndpointMethods,
		yacyproto.ParseMessageRequest,
		acceptingEndpoint{
			identity:  identity,
			allowance: newSenderAllowance(now),
			inbox:     inbox,
		}.Serve,
	)
}

func MountRefusing(router httpguard.WireRouter) {
	httpguard.MountFields(
		router,
		yacyproto.PathMessage,
		yacyproto.MessageEndpointMethods,
		yacyproto.ParseMessageRequest,
		refusingEndpoint{}.Serve,
	)
}
