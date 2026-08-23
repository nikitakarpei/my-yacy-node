package rwiingress

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type Config struct {
	PostingCap int
	Pause      time.Duration
	Refusals   rwiadmission.RefusalObserver
}

func Mount(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	receiver rwiadmission.PostingReceiver,
	config Config,
) {
	httpguard.Mount(
		router,
		yacyproto.PathTransferRWI,
		yacyproto.TransferRWIEndpointMethods,
		yacyproto.ParseTransferRWIRequest,
		transferRWIEndpoint{
			identity:   identity,
			intake:     receiver,
			postingCap: config.PostingCap,
			pause:      config.Pause,
			refusals:   config.Refusals,
		}.Serve,
	)
}
