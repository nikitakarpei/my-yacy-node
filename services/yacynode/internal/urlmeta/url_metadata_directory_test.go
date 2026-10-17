package urlmeta_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
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

	return yacymodel.URLMetadata{Hash: urlHashOf(t, address), Address: address}
}

func urlHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func writeAll(t *testing.T, v *vault.Vault, module urlPorts, metadata ...yacymodel.URLMetadata) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		for _, written := range metadata {
			if _, err := module.Writer.Write(
				tx,
				written,
				urlmeta.SourcePeer(sendingNetwork),
			); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func currentVersionsOf(
	t *testing.T,
	v *vault.Vault,
	directory urlmeta.URLDirectory,
	hashes ...yacymodel.URLHash,
) map[yacymodel.URLHash]urlmeta.CurrentVersion {
	t.Helper()

	var versions map[yacymodel.URLHash]urlmeta.CurrentVersion
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		read, err := directory.CurrentVersionsOf(tx, hashes)
		versions = read

		return err
	}); err != nil {
		t.Fatalf("CurrentVersionsOf: %v", err)
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
		[]yacymodel.URLHash{row.Hash, urlHashOf(t, "http://example.com/absent")},
	)
	if len(rows) != 1 || rows[row.Hash].Hash != row.Hash ||
		storedURLCount(t, v, module.Directory) != 1 {
		t.Fatalf("MetadataPerHash = %v, want one matching row and no absent one", rows)
	}
}

func TestStoredMetadataKnowsItsSource(t *testing.T) {
	v, module := openObservedModule(t)
	sent, crawled := urlMetadata(t, "a"), urlMetadata(t, "b")
	writeAll(t, v, module, sent)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := module.Writer.Write(tx, crawled, urlmeta.SourceThisNode())

		return err
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	versions := currentVersionsOf(t, v, module.Directory, sent.Hash, crawled.Hash)
	if versions[sent.Hash].Source != urlmeta.SourcePeer(sendingNetwork) ||
		versions[crawled.Hash].Source != urlmeta.SourceThisNode() {
		t.Fatalf("versions = %+v, want the peer network and this node", versions)
	}
}
