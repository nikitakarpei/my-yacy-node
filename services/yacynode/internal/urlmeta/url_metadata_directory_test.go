package urlmeta_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var sendingNetwork = rwiorigin.Of("203.0.113.7")

func localIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:         yacymodel.WordHash("self"),
		NetworkName:  "freeworld",
		Capabilities: yacymodel.PeerCapabilities{AcceptRemoteIndex: true},
	}
}

type urlPorts struct {
	Directory urlmeta.URLDirectory
	Evictor   urlmeta.URLEvictor
	Writer    urlmeta.URLMetadataWriter
}

func urlMetadata(t *testing.T, seed string) yacymodel.URLMetadata {
	t.Helper()

	address := "http://example.com/" + seed

	return yacymodel.URLMetadata{Hash: urlhashtest.URLHashOf(t, address), Address: address}
}

func writeAll(t *testing.T, v *vault.Vault, module urlPorts, metadata ...yacymodel.URLMetadata) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		for _, written := range metadata {
			if err := module.Writer.Write(
				tx,
				written,
				pageversion.SourcePeer(sendingNetwork),
			); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func versionsOf(
	t *testing.T,
	v *vault.Vault,
	directory urlmeta.URLDirectory,
	hashes ...yacymodel.URLHash,
) map[yacymodel.URLHash]pageversion.Version {
	t.Helper()

	var versions map[yacymodel.URLHash]pageversion.Version
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		read, err := directory.VersionsOf(tx, hashes)
		versions = read

		return err
	}); err != nil {
		t.Fatalf("VersionsOf: %v", err)
	}

	return versions
}

func TestWrittenMetadataReadsBackByItsHash(t *testing.T) {
	v, module := openObservedModule(t)
	row := urlMetadata(t, "a")

	writeAll(t, v, module, row)

	rows := metadataPerHash(
		t,
		v,
		module.Directory,
		[]yacymodel.URLHash{row.Hash, urlhashtest.URLHashOf(t, "http://example.com/absent")},
	)
	if len(rows) != 1 || rows[row.Hash].Hash != row.Hash ||
		storedURLCount(t, v, module.Directory) != 1 {
		t.Fatalf("MetadataPerHash = %v, want one matching row and no absent one", rows)
	}
}

func TestStoredMetadataKnowsItsVersionAndSource(t *testing.T) {
	v, module := openObservedModule(t)
	sent, crawled := urlMetadata(t, "a"), urlMetadata(t, "b")
	sent.Title = "Hello World"
	writeAll(t, v, module, sent)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return module.Writer.Write(tx, crawled, pageversion.SourceThisNode())
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	versions := versionsOf(t, v, module.Directory, sent.Hash, crawled.Hash)
	if versions[sent.Hash] != pageversion.FromMetadata(
		sent,
		pageversion.SourcePeer(sendingNetwork),
	) ||
		versions[crawled.Hash] != pageversion.FromMetadata(crawled, pageversion.SourceThisNode()) {
		t.Fatalf("versions = %+v, want the peer network and this node", versions)
	}
}
