package rwiingress

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func Mount(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	receiver rwiadmission.PostingReceiver,
) {
	httpguard.Mount(
		router,
		yacyproto.PathTransferRWI,
		yacyproto.TransferRWIEndpointMethods,
		yacyproto.ParseTransferRWIRequest,
		transferRWIEndpoint{identity: identity, intake: receiver}.Serve,
	)
}
