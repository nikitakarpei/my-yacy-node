package remoteindexingress

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func Mount(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	postings remoteindexadmission.PostingReceiver,
	urlMetadata remoteindexadmission.URLMetadataReceiver,
) {
	httpguard.MountFields(
		router,
		yacyproto.PathTransferRWI,
		yacyproto.TransferRWIEndpointMethods,
		yacyproto.ParseTransferRWIRequest,
		transferRWIEndpoint{identity: identity, receiver: postings}.Serve,
	)
	httpguard.MountFields(
		router,
		yacyproto.PathTransferURL,
		yacyproto.TransferURLEndpointMethods,
		yacyproto.ParseTransferURLRequest,
		transferURLEndpoint{identity: identity, receiver: urlMetadata}.Serve,
	)
}
