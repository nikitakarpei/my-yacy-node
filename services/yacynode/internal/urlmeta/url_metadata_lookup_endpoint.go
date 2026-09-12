package urlmeta

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type urlMetadataLookupEndpoint struct {
	identity                    nodeidentity.Identity
	vault                       *vault.Vault
	directory                   URLDirectory
	servedURLMetadataPerRequest int
}

func (e urlMetadataLookupEndpoint) Serve(
	ctx context.Context,
	req yacyproto.URLMetadataRequest,
) (yacyproto.URLMetadataResponse, error) {
	if !e.identity.NetworkMatches(req.NetworkName) {
		slog.DebugContext(ctx, "url metadata lookup refused",
			slog.String("reason", "network mismatch"),
			slog.String("networkName", req.NetworkName),
		)

		return yacyproto.URLMetadataResponse{}, nil
	}

	servedMetadata, err := e.storedMetadataOf(ctx, e.askedURLsOf(req))
	if err != nil {
		return yacyproto.URLMetadataResponse{}, fmt.Errorf("read url metadata: %w", err)
	}

	slog.DebugContext(ctx, "url metadata lookup served",
		slog.Int("askedUrlCount", len(req.URLs)),
		slog.Int("servedUrlCount", len(servedMetadata)),
	)

	return yacyproto.URLMetadataResponse{URLs: servedMetadata}, nil
}

func (e urlMetadataLookupEndpoint) askedURLsOf(
	req yacyproto.URLMetadataRequest,
) []yacymodel.URLHash {
	return req.URLs[:min(len(req.URLs), e.servedURLMetadataPerRequest)]
}

func (e urlMetadataLookupEndpoint) storedMetadataOf(
	ctx context.Context,
	askedURLs []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	var metadataPerHash map[yacymodel.URLHash]yacymodel.URLMetadata
	err := e.vault.View(ctx, func(tx *vault.Txn) error {
		stored, err := e.directory.MetadataPerHash(tx, askedURLs)
		metadataPerHash = stored

		return err
	})
	if err != nil {
		return nil, err
	}

	storedMetadata := make([]yacymodel.URLMetadata, 0, len(metadataPerHash))
	for _, hash := range askedURLs {
		if metadata, stored := metadataPerHash[hash]; stored {
			storedMetadata = append(storedMetadata, metadata)
		}
	}

	return storedMetadata, nil
}
