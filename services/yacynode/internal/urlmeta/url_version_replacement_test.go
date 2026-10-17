package urlmeta_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

func modifiedOn(metadata yacymodel.URLMetadata, day int) yacymodel.URLMetadata {
	metadata.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, day))

	return metadata
}

func receiveFrom(
	t *testing.T,
	module urlPorts,
	sender rwiorigin.Origin,
	metadata ...yacymodel.URLMetadata,
) {
	t.Helper()

	if _, err := module.Receiver.Receive(context.Background(), sender, metadata); err != nil {
		t.Fatalf("Receive: %v", err)
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

func TestIntakeReportsOneOutcomePerRecord(t *testing.T) {
	_, module := openObservedModule(t)
	stored := modifiedOn(urlMetadata(t, "a"), 10)
	older := modifiedOn(stored, 9)
	sameDay := stored
	sameDay.LocalLinks = 9
	postdated := modifiedOn(urlMetadata(t, "b"), 17)

	receiveFrom(t, module, sendingNetwork, stored)
	receiveFrom(t, module, sendingNetwork, stored, older, sameDay, postdated)

	want := recordedOutcomes{
		urlmeta.OutcomeStoredNewURL:         1,
		urlmeta.OutcomeDroppedAlreadyStored: 1,
		urlmeta.OutcomeDroppedOtherVersion:  2,
		urlmeta.OutcomeDroppedFutureDate:    1,
	}
	if !maps.Equal(module.Outcomes, want) {
		t.Fatalf("outcomes = %v, want %v", module.Outcomes, want)
	}
}

func TestIntakeReplacesALaterVersionTogetherWithEveryPostingOfItsURL(t *testing.T) {
	observer := &recordingObserver{}
	v, module := openObservedModule(t, observer)
	earlier := modifiedOn(urlMetadata(t, "a"), 9)
	later := modifiedOn(earlier, 10)
	later.Title = "later"

	receiveFrom(t, module, sendingNetwork, earlier)
	receiveFrom(t, module, sendingNetwork, later)

	if want := []string{"stored", "purged", "stored"}; !slices.Equal(observer.events, want) {
		t.Fatalf(
			"events = %v, want the url purged before the later version is stored",
			observer.events,
		)
	}
	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{later.Hash})
	if rows[later.Hash].Title != "later" ||
		module.Outcomes[urlmeta.OutcomeReplacedOlderVersion] != 1 {
		t.Fatalf("stored = %+v, outcomes = %v, want the later version", rows, module.Outcomes)
	}
}

func TestIntakeSwapsPerNetworkStopAtTheHourlyLimit(t *testing.T) {
	clock := intakeClock()
	v, module := openModuleAt(t, func() time.Time { return clock })
	metadata := urlMetadata(t, "a")
	modifiedDay := func() int {
		return currentVersionsOf(t, v, module.Directory, metadata.Hash)[metadata.Hash].
			Fingerprint.Modified.Time().Day()
	}

	receiveFrom(t, module, sendingNetwork, modifiedOn(metadata, 1))
	receiveFrom(t, module, sendingNetwork, modifiedOn(metadata, 2))
	receiveFrom(t, module, sendingNetwork, modifiedOn(metadata, 3))
	if modifiedDay() != 2 || module.Outcomes[urlmeta.OutcomeDroppedOverSwapLimit] != 1 {
		t.Fatalf("modified day = %d, outcomes = %v, want the second swap refused",
			modifiedDay(), module.Outcomes)
	}

	receiveFrom(t, module, otherNetwork, modifiedOn(metadata, 3))
	clock = clock.Add(time.Hour)
	receiveFrom(t, module, sendingNetwork, modifiedOn(metadata, 4))

	if modifiedDay() != 4 || module.Outcomes[urlmeta.OutcomeReplacedOlderVersion] != 3 {
		t.Fatalf(
			"modified day = %d, outcomes = %v, want swaps from another network and an hour later",
			modifiedDay(),
			module.Outcomes,
		)
	}
}

func TestIntakeSwapsInOneTransferStopAtTheHourlyLimit(t *testing.T) {
	_, module := openObservedModule(t)
	first, second := urlMetadata(t, "a"), urlMetadata(t, "b")
	receiveFrom(t, module, sendingNetwork, modifiedOn(first, 1), modifiedOn(second, 1))

	receiveFrom(t, module, sendingNetwork, modifiedOn(first, 2), modifiedOn(second, 2))

	if module.Outcomes[urlmeta.OutcomeReplacedOlderVersion] != 1 ||
		module.Outcomes[urlmeta.OutcomeDroppedOverSwapLimit] != 1 {
		t.Fatalf("outcomes = %v, want one swap and one over the limit", module.Outcomes)
	}
}

func TestStoredMetadataKnowsItsSource(t *testing.T) {
	v, module := openObservedModule(t)
	sent, crawled := urlMetadata(t, "a"), urlMetadata(t, "b")
	receiveFrom(t, module, sendingNetwork, sent)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return module.Admitter.Admit(tx, crawled)
	}); err != nil {
		t.Fatalf("Admit: %v", err)
	}

	versions := currentVersionsOf(t, v, module.Directory, sent.Hash, crawled.Hash)
	if versions[sent.Hash].Source != urlmeta.SourcePeer(sendingNetwork) ||
		versions[crawled.Hash].Source != urlmeta.SourceThisNode() {
		t.Fatalf("versions = %+v, want the peer network and this node", versions)
	}
}

type rawRecordCodec struct{}

func (rawRecordCodec) Encode(raw []byte) ([]byte, error) { return raw, nil }

func (rawRecordCodec) Decode(raw []byte) ([]byte, error) { return slices.Clone(raw), nil }

func TestMetadataStoredBeforeSourcesHasNeitherSource(t *testing.T) {
	ctx := context.Background()
	engine := memoryvault.OpenEngine(0)
	v, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	directory, _, admitter, _, err := urlmeta.Open(
		ctx, v, intakeClock, urlmeta.SwapLimit{}, recordedOutcomes{}, nil,
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	crawled := urlMetadata(t, "a")
	if err := v.Update(
		ctx,
		func(tx *vault.Txn) error { return admitter.Admit(tx, crawled) },
	); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	stripSourceOf(t, engine, crawled.Hash)

	source := currentVersionsOf(t, v, directory, crawled.Hash)[crawled.Hash].Source
	if source == urlmeta.SourceThisNode() || source == urlmeta.SourcePeer(sendingNetwork) {
		t.Fatalf("source = %+v, want neither this node nor a peer", source)
	}
}

func stripSourceOf(t *testing.T, engine vault.Engine, url yacymodel.URLHash) {
	t.Helper()

	raw, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	records, err := raw.RegisterCollection(
		"urlmeta",
		vault.SingleKey(hashkeypart.URLHash).KeyLayout(),
		rawRecordCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}
	if err := raw.Update(context.Background(), func(tx *vault.Txn) error {
		record, _, err := records.Get(tx, url)
		if err != nil {
			return err
		}
		_, err = records.Put(tx, url, record[:len(record)-1])

		return err
	}); err != nil {
		t.Fatalf("strip source: %v", err)
	}
}
